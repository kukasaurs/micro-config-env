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
	tt := timeTransformer{override: options.Override}
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
			kv := strings.FieldsFunc(nval, func(c rune) bool { return c == '=' })
			mkey := reflect.Indirect(reflect.New(kt))
			mval := reflect.Indirect(reflect.New(et))
			if err := fillValue(ctx, mkey, kv[0]); err != nil {
				return err
			}
			if err := fillValue(ctx, mval, kv[1]); err != nil {
				return err
			}
			value.SetMapIndex(mkey, mval)
		}
	case reflect.Slice, reflect.Array:
		nvals := strings.FieldsFunc(val, func(c rune) bool { return c == ',' || c == ';' })
		value.Set(reflect.MakeSlice(reflect.SliceOf(value.Type().Elem()), len(nvals), len(nvals)))
		for idx, nval := range nvals {
			nvalue := reflect.Indirect(reflect.New(value.Type().Elem()))
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
		value.Set(reflect.ValueOf(v))
	case reflect.String:
		value.Set(reflect.ValueOf(val))
	case reflect.Float32:
		v, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(float32(v)))
	case reflect.Float64:
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(float64(v)))
	case reflect.Int:
		v, err := strconv.ParseInt(val, 10, 0)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(int(v)))
	case reflect.Int8:
		v, err := strconv.ParseInt(val, 10, 8)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(int8(v)))
	case reflect.Int16:
		v, err := strconv.ParseInt(val, 10, 16)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(int16(v)))
	case reflect.Int32:
		v, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(int32(v)))
	case reflect.Int64:
		if value.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("cannot parse duration %q: %w", val, err)
			}
			value.SetInt(int64(d))
			return nil
		}
		v, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(int64(v)))
	case reflect.Uint:
		v, err := strconv.ParseUint(val, 10, 0)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(uint(v)))
	case reflect.Uint8:
		v, err := strconv.ParseUint(val, 10, 8)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(uint8(v)))
	case reflect.Uint16:
		v, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(uint16(v)))
	case reflect.Uint32:
		v, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(uint32(v)))
	case reflect.Uint64:
		v, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(uint64(v)))
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
		for _, tag := range strings.Split(tags, ",") {
			if err := os.Setenv(tag, fmt.Sprintf("%v", value.Interface())); err != nil && !c.opts.AllowFail {
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
			if value.Type() == reflect.TypeOf(time.Time{}) {
				if eval, ok := getEnvValue(field, structTag); ok {
					parsed, err := time.Parse(time.RFC3339, eval)
					if err != nil {
						return fmt.Errorf("cannot parse time.Time %q: %w", eval, err)
					}
					value.Set(reflect.ValueOf(parsed))
				}
				continue
			}
			value.Set(reflect.Indirect(reflect.New(value.Type())))
			if err := fillValues(ctx, value, structTag); err != nil {
				return err
			}
			continue
		case reflect.Ptr:
			if value.Type().Elem() == reflect.TypeOf(time.Time{}) {
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

type timeTransformer struct {
	override bool
}

func (t timeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf(time.Time{}) {
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
	if typ == reflect.TypeOf((*time.Time)(nil)) {
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
	return nil
}
