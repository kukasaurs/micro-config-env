package env

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"dario.cat/mergo"

	"go.unistack.org/micro/v3/config"
	rutil "go.unistack.org/micro/v3/util/reflect"
)

var DefaultStructTag = "env"

type envConfig struct {
	opts config.Options
}

func (c *envConfig) Options() config.Options {
	return c.opts
}

func (c *envConfig) Init(opts ...config.Option) error {
	for _, o := range opts {
		o(&c.opts)
	}

	if err := config.DefaultBeforeInit(c.opts.Context, c); err != nil && !c.opts.AllowFail {
		return err
	}

	if err := config.DefaultAfterInit(c.opts.Context, c); err != nil && !c.opts.AllowFail {
		return err
	}

	return nil
}

func (c *envConfig) Load(ctx context.Context, opts ...config.LoadOption) error {
	if c.opts.SkipLoad != nil && c.opts.SkipLoad(ctx, c) {
		return nil
	}

	if err := config.DefaultBeforeLoad(ctx, c); err != nil && !c.opts.AllowFail {
		return err
	}

	options := config.NewLoadOptions(opts...)
	mopts := []func(*mergo.Config){mergo.WithTypeCheck}
	tt := typeTransformer{override: options.Override}
	mopts = append(mopts, mergo.WithTransformers(tt))
	if options.Override {
		mopts = append(mopts, mergo.WithOverride)
	}
	if options.Append {
		mopts = append(mopts, mergo.WithAppendSlice)
	}

	dst := c.opts.Struct
	if options.Struct != nil {
		dst = options.Struct
	}

	src, err := rutil.Zero(dst)
	if err == nil {
		if err = fillValues(ctx, reflect.ValueOf(src), c.opts.StructTag); err == nil {
			err = mergo.Merge(dst, src, mopts...)
		}
	}

	if err != nil && !c.opts.AllowFail {
		return err
	}

	if err := config.DefaultAfterLoad(ctx, c); err != nil && !c.opts.AllowFail {
		return err
	}

	return nil
}

func fillValue(ctx context.Context, value reflect.Value, val string) error {
	switch value.Kind() {
	case reflect.Map:
		t := value.Type()
		nvals := strings.FieldsFunc(val, func(c rune) bool { return c == ',' || c == ';' })
		if value.IsNil() {
			value.Set(reflect.MakeMapWithSize(t, len(nvals)))
		}
		kt := t.Key()
		et := t.Elem()
		for _, nval := range nvals {
			kv := strings.SplitN(nval, "=", 2)
			if len(kv) != 2 {
				return fmt.Errorf("invalid map entry %q", nval)
			}
			mkey := reflect.New(kt).Elem()
			mval := reflect.New(et).Elem()
			if err := fillValue(ctx, mkey, kv[0]); err != nil {
				return err
			}
			if err := fillValue(ctx, mval, kv[1]); err != nil {
				return err
			}
			value.SetMapIndex(mkey, mval)
		}
	case reflect.Slice:
		t := value.Type()
		nvals := strings.FieldsFunc(val, func(c rune) bool { return c == ',' || c == ';' })
		value.Set(reflect.MakeSlice(t, len(nvals), len(nvals)))
		for idx, nval := range nvals {
			nvalue := reflect.New(t.Elem()).Elem()
			if err := fillValue(ctx, nvalue, nval); err != nil {
				return err
			}
			value.Index(idx).Set(nvalue)
		}
	case reflect.Array:
		t := value.Type()
		nvals := strings.FieldsFunc(val, func(c rune) bool { return c == ',' || c == ';' })
		if len(nvals) != t.Len() {
			return fmt.Errorf("array length mismatch: got %d, want %d", len(nvals), t.Len())
		}
		for idx, nval := range nvals {
			nvalue := reflect.New(t.Elem()).Elem()
			if err := fillValue(ctx, nvalue, nval); err != nil {
				return err
			}
			value.Index(idx).Set(nvalue)
		}
	case reflect.Bool:
		v, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		value.SetBool(v)
	case reflect.String:
		value.SetString(val)
	case reflect.Float32:
		v, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return err
		}
		value.SetFloat(v)
	case reflect.Float64:
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		value.SetFloat(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if isDurationType(value.Type()) {
			d, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("cannot parse duration %q: %w", val, err)
			}
			value.SetInt(int64(d))
			return nil
		}
		bitSize := value.Type().Bits()
		v, err := strconv.ParseInt(val, 10, bitSize)
		if err != nil {
			return err
		}
		value.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bitSize := value.Type().Bits()
		v, err := strconv.ParseUint(val, 10, bitSize)
		if err != nil {
			return err
		}
		value.SetUint(v)
	}
	return nil
}

func (c *envConfig) setValues(ctx context.Context, valueOf reflect.Value) error {
	var values reflect.Value

	if valueOf.Kind() == reflect.Ptr {
		values = valueOf.Elem()
	} else {
		values = valueOf
	}

	if values.Kind() == reflect.Invalid {
		return config.ErrInvalidStruct
	}

	fields := values.Type()

	for idx := 0; idx < fields.NumField(); idx++ {
		field := fields.Field(idx)
		value := values.Field(idx)
		if rutil.IsZero(value) {
			continue
		}
		if !field.IsExported() {
			continue
		}
		switch value.Kind() {
		case reflect.Struct:
			if err := c.setValues(ctx, value); err != nil {
				return err
			}
			continue
		case reflect.Ptr:
			value = value.Elem()
			if err := c.setValues(ctx, value); err != nil {
				return err
			}
			continue
		}

		tags, ok := field.Tag.Lookup(c.opts.StructTag)
		if !ok {
			continue
		}
		var strVal string
		switch value.Kind() {
		case reflect.Slice, reflect.Array:
			parts := make([]string, value.Len())
			for i := 0; i < value.Len(); i++ {
				parts[i] = fmt.Sprintf("%v", value.Index(i).Interface())
			}
			strVal = strings.Join(parts, ",")
		case reflect.Map:
			parts := make([]string, 0, value.Len())
			for _, key := range value.MapKeys() {
				parts = append(parts, fmt.Sprintf("%v=%v", key.Interface(), value.MapIndex(key).Interface()))
			}
			strVal = strings.Join(parts, ",")
		default:
			strVal = fmt.Sprintf("%v", value.Interface())
		}
		for _, tag := range strings.Split(tags, ",") {
			if err := os.Setenv(tag, strVal); err != nil && !c.opts.AllowFail {
				return err
			}
		}
	}

	return nil
}

func getEnvValue(field reflect.StructField, structTag string) (string, bool) {
	tags, ok := field.Tag.Lookup(structTag)
	if !ok {
		return "", false
	}
	var val string
	for _, tag := range strings.Split(tags, ",") {
		if v, ok := os.LookupEnv(tag); ok {
			val = v
		}
	}
	return val, val != ""
}

func fillValues(ctx context.Context, valueOf reflect.Value, structTag string) error {
	var values reflect.Value

	if valueOf.Kind() == reflect.Ptr {
		values = valueOf.Elem()
	} else {
		values = valueOf
	}

	if values.Kind() == reflect.Invalid {
		return config.ErrInvalidStruct
	}

	fields := values.Type()

	for idx := 0; idx < fields.NumField(); idx++ {
		field := fields.Field(idx)
		if !field.IsExported() {
			continue
		}
		value := values.Field(idx)
		if !value.CanSet() {
			continue
		}

		switch value.Kind() {
		case reflect.Struct:
			if isTimeType(value.Type()) {
				if eval, ok := getEnvValue(field, structTag); ok {
					parsed, err := time.Parse(time.RFC3339, eval)
					if err != nil {
						return fmt.Errorf("cannot parse time.Time %q: %w", eval, err)
					}
					value.Set(reflect.ValueOf(parsed))
				}
				continue
			}
			if err := fillValues(ctx, value, structTag); err != nil {
				return err
			}
			continue
		case reflect.Ptr:
			if isTimeType(value.Type().Elem()) {
				if eval, ok := getEnvValue(field, structTag); ok {
					parsed, err := time.Parse(time.RFC3339, eval)
					if err != nil {
						return fmt.Errorf("cannot parse *time.Time %q: %w", eval, err)
					}
					value.Set(reflect.ValueOf(&parsed))
				}
				continue
			}
			if value.IsNil() {
				if value.Type().Elem().Kind() != reflect.Struct {
					// nil pointer to a non-struct: leave it alone
					break
				}
				// nil pointer to struct: create a zero instance
				value.Set(reflect.New(value.Type().Elem()))
			}
			if err := fillValues(ctx, value.Elem(), structTag); err != nil {
				return err
			}
			continue
		default:
			if eval, ok := getEnvValue(field, structTag); ok {
				if err := fillValue(ctx, value, eval); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *envConfig) Save(ctx context.Context, opts ...config.SaveOption) error {
	if c.opts.SkipSave != nil && c.opts.SkipSave(ctx, c) {
		return nil
	}

	options := config.NewSaveOptions(opts...)

	if err := config.DefaultBeforeSave(ctx, c); err != nil && !c.opts.AllowFail {
		return err
	}

	dst := c.opts.Struct
	if options.Struct != nil {
		dst = options.Struct
	}

	if err := c.setValues(ctx, reflect.ValueOf(dst)); err != nil && !c.opts.AllowFail {
		return err
	}

	if err := config.DefaultAfterSave(ctx, c); err != nil && !c.opts.AllowFail {
		return err
	}

	return nil
}

func (c *envConfig) String() string {
	return "env"
}

func (c *envConfig) Name() string {
	return c.opts.Name
}

func (c *envConfig) Watch(ctx context.Context, opts ...config.WatchOption) (config.Watcher, error) {
	w := &envWatcher{
		opts:  c.opts,
		wopts: config.NewWatchOptions(opts...),
		done:  make(chan struct{}),
		vchan: make(chan map[string]interface{}),
		echan: make(chan error),
	}

	go w.run()

	return w, nil
}

func NewConfig(opts ...config.Option) config.Config {
	options := config.NewOptions(opts...)
	if len(options.StructTag) == 0 {
		options.StructTag = DefaultStructTag
	}
	return &envConfig{opts: options}
}

type typeTransformer struct {
	override bool
}

func (t typeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if isTimeType(typ) {
		return func(dst, src reflect.Value) error {
			if !dst.CanSet() || src.IsZero() {
				return nil
			}
			if !t.override && !dst.IsZero() {
				return nil
			}
			dst.Set(src)
			return nil
		}
	}
	if isTimePtrType(typ) {
		return func(dst, src reflect.Value) error {
			if !dst.CanSet() || src.IsNil() {
				return nil
			}
			if src.Elem().IsZero() {
				return nil
			}
			if !t.override && !dst.IsNil() && !dst.Elem().IsZero() {
				return nil
			}
			dst.Set(src)
			return nil
		}
	}
	if typ.Kind() == reflect.Array {
		return func(dst, src reflect.Value) error {
			if !dst.CanSet() || src.IsZero() {
				return nil
			}
			if !t.override && !isZeroArray(dst) {
				return nil
			}
			dst.Set(src)
			return nil
		}
	}

	return nil
}
