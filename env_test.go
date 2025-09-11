package env

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.unistack.org/micro/v3/config"
	rutil "go.unistack.org/micro/v3/util/reflect"
)

type Config struct {
	StringValue    string            `env:"STRING_VALUE,STRING_VALUE2"`
	BoolValue      bool              `env:"BOOL_VALUE"`
	StringSlice    []string          `env:"STRING_SLICE"`
	IntSlice       []int             `env:"INT_SLICE"`
	MapStringValue map[string]string `env:"MAP_STRING"`
	MapIntValue    map[string]int    `env:"MAP_INT"`
}

func TestMerge(t *testing.T) {
	defer func() {
		for _, v := range []string{"STRING_VALUE", "BOOL_VALUE", "STRING_SLICE", "INT_SLICE", "MAP_STRING", "MAP_INT"} {
			if err := os.Unsetenv(v); err != nil {
				t.Fatal(err)
			}
		}
	}()

	ctx := context.Background()
	type Nested struct {
		Name string `env:"NAME_VALUE"`
	}
	type Cfg struct {
		Name   string `env:"NAME_VALUE"`
		Nested Nested
	}

	conf := &Cfg{}

	cfg := NewConfig(config.Struct(conf))

	if err := cfg.Init(); err != nil {
		t.Fatal(err)
	}

	if err := cfg.Load(ctx, config.LoadOverride(true), config.LoadAppend(true)); err != nil {
		t.Fatal(err)
	}

	w, err := cfg.Watch(ctx, config.WatchInterval(50*time.Millisecond, 500*time.Millisecond))
	defer func() {
		if err := w.Stop(); err != nil {
			t.Fatal(err)
		}
	}()
	if err != nil {
		t.Fatal(err)
	}

	require.NoError(t, os.Setenv("NAME_VALUE", "after"))
	changes, err := w.Next()
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range changes {
		if err := rutil.SetFieldByPath(conf, v, k); err != nil {
			t.Fatal(err)
		}
	}

	if conf.Name != "after" || conf.Nested.Name != "after" {
		t.Fatalf("changes %#+v not applied to %#+v", changes, conf)
	}
}

func TestLoad(t *testing.T) {
	defer func() {
		for _, v := range []string{"STRING_VALUE", "STRING_VALUE2", "BOOL_VALUE", "STRING_SLICE", "INT_SLICE", "MAP_STRING", "MAP_INT"} {
			if err := os.Unsetenv(v); err != nil {
				t.Fatal(err)
			}
		}
	}()

	ctx := context.Background()
	conf := &Config{StringValue: "before_load"}
	cfg := NewConfig(config.Struct(conf))

	if err := cfg.Init(); err != nil {
		t.Fatal(err)
	}

	if err := cfg.Load(ctx, config.LoadOverride(true), config.LoadAppend(true)); err != nil {
		t.Fatal(err)
	}

	if conf.StringValue != "before_load" {
		t.Fatalf("something wrong with env config: %#+v", conf)
	}

	require.NoError(t, os.Setenv("STRING_VALUE", "STRING_VALUE"))
	require.NoError(t, os.Setenv("BOOL_VALUE", "true"))
	require.NoError(t, os.Setenv("STRING_SLICE", "STRING_SLICE1,STRING_SLICE2;STRING_SLICE3"))
	require.NoError(t, os.Setenv("INT_SLICE", "1,2,3,4,5"))
	require.NoError(t, os.Setenv("MAP_STRING", "key1=val1,key2=val2"))
	require.NoError(t, os.Setenv("MAP_INT", "key1=1,key2=2"))

	if err := cfg.Load(ctx, config.LoadOverride(true), config.LoadAppend(true)); err != nil {
		t.Fatal(err)
	}
	if conf.StringValue != "STRING_VALUE" {
		t.Fatalf("something wrong with env config: %#+v", conf)
	}

	if !conf.BoolValue {
		t.Fatalf("something wrong with env config: %#+v", conf)
	}

	if len(conf.StringSlice) != 3 {
		t.Fatalf("something wrong with env config: %#+v", conf.StringSlice)
	}

	if len(conf.MapStringValue) != 2 {
		t.Fatalf("something wrong with env config: %#+v", conf.MapStringValue)
	}

	if len(conf.MapIntValue) != 2 {
		t.Fatalf("something wrong with env config: %#+v", conf.MapIntValue)
	}

	for _, v := range []string{"STRING_VALUE", "STRING_VALUE2", "BOOL_VALUE", "STRING_SLICE", "INT_SLICE", "MAP_STRING", "MAP_INT"} {
		if err := os.Unsetenv(v); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSave(t *testing.T) {
	defer func() {
		for _, v := range []string{"STRING_VALUE", "STRING_VALUE2", "BOOL_VALUE", "STRING_SLICE", "INT_SLICE", "MAP_STRING", "MAP_INT"} {
			if err := os.Unsetenv(v); err != nil {
				t.Fatal(err)
			}
		}
	}()

	ctx := context.Background()
	conf := &Config{StringValue: "MICRO_CONFIG_ENV"}
	cfg := NewConfig(config.Struct(conf))

	if err := cfg.Init(); err != nil {
		t.Fatal(err)
	}

	if _, ok := os.LookupEnv("STRING_VALUE"); ok {
		if err := os.Unsetenv("STRING_VALUE"); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.Unsetenv("STRING_VALUE"); err != nil {
		t.Fatal(err)
	}

	if err := cfg.Save(ctx); err != nil {
		t.Fatal(err)
	}

	if _, ok := os.LookupEnv("STRING_VALUE"); !ok {
		t.Fatal("env value STRING_VALUE=MICRO_CONFIG_ENV not set")
	}

	if err := os.Unsetenv("STRING_VALUE"); err != nil {
		t.Fatal(err)
	}

	for _, tv := range []string{"STRING_VALUE", "BOOL_VALUE", "STRING_SLICE", "INT_SLICE", "MAP_STRING", "MAP_INT"} {
		if v, ok := os.LookupEnv("STRING_VALUE"); ok {
			t.Fatalf("env value %s=%s set", tv, v)
		}
	}

	for _, v := range []string{"STRING_VALUE", "STRING_VALUE2", "BOOL_VALUE", "STRING_SLICE", "INT_SLICE", "MAP_STRING", "MAP_INT"} {
		if err := os.Unsetenv(v); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadMultiple(t *testing.T) {
	defer func() {
		for _, v := range []string{"STRING_VALUE", "STRING_VALUE2", "BOOL_VALUE", "STRING_SLICE", "INT_SLICE", "MAP_STRING", "MAP_INT"} {
			if err := os.Unsetenv(v); err != nil {
				t.Fatal(err)
			}
		}
	}()

	ctx := context.Background()
	conf := &Config{StringValue: "before_load"}
	cfg := NewConfig(config.Struct(conf))

	if err := cfg.Init(); err != nil {
		t.Fatal(err)
	}

	if err := cfg.Load(ctx, config.LoadOverride(true), config.LoadAppend(true)); err != nil {
		t.Fatal(err)
	}

	if conf.StringValue != "before_load" {
		t.Fatalf("something wrong with env config: %#+v", conf)
	}

	require.NoError(t, os.Setenv("STRING_VALUE", "STRING_VALUE1"))
	require.NoError(t, os.Setenv("STRING_VALUE2", "STRING_VALUE2"))
	defer func() {
		for _, v := range []string{"STRING_VALUE", "STRING_VALUE2"} {
			if err := os.Unsetenv(v); err != nil {
				t.Fatal(err)
			}
		}
	}()

	if err := cfg.Load(ctx, config.LoadOverride(true), config.LoadAppend(true)); err != nil {
		t.Fatal(err)
	}
	if conf.StringValue != "STRING_VALUE2" {
		t.Fatalf("something wrong with env config: %#+v", conf)
	}

	for _, v := range []string{"STRING_VALUE", "STRING_VALUE2", "BOOL_VALUE", "STRING_SLICE", "INT_SLICE", "MAP_STRING", "MAP_INT"} {
		if err := os.Unsetenv(v); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEnv_SupportedTypes(t *testing.T) {
	type Config struct {
		IntValue   int   `env:"INT_VALUE"`
		Int8Value  int8  `env:"INT8_VALUE"`
		Int16Value int16 `env:"INT16_VALUE"`
		Int32Value int32 `env:"INT32_VALUE"`
		Int64Value int64 `env:"INT64_VALUE"`

		UintValue   uint   `env:"UINT_VALUE"`
		Uint8Value  uint8  `env:"UINT8_VALUE"`
		Uint16Value uint16 `env:"UINT16_VALUE"`
		Uint32Value uint32 `env:"UINT32_VALUE"`
		Uint64Value uint64 `env:"UINT64_VALUE"`

		Float32Value float32 `env:"FLOAT32_VALUE"`
		Float64Value float64 `env:"FLOAT64_VALUE"`

		BoolValue bool `env:"BOOL_VALUE"`

		StringValue string `env:"STRING_VALUE"`

		StringSlice []string `env:"STRING_SLICE"`
		IntSlice    []int    `env:"INT_SLICE"`

		MapStringValue map[string]string `env:"MAP_STRING"`
		MapIntValue    map[string]int    `env:"MAP_INT"`

		DurationValue time.Duration `env:"DURATION_VALUE"`
		TimeValue     time.Time     `env:"TIME_VALUE"`
		TimePtrValue  *time.Time    `env:"TIME_PTR_VALUE"`

		StringArray [3]string `env:"STRING_ARRAY"`
		IntArray    [5]int    `env:"INT_ARRAY"`
	}

	tests := []struct {
		name   string
		envVar string
		envVal string
		want   func() *Config
	}{
		// integers
		{
			name:   "int type",
			envVar: "INT_VALUE",
			envVal: "100",
			want:   func() *Config { return &Config{IntValue: 100} },
		},
		{
			name:   "int8 type",
			envVar: "INT8_VALUE",
			envVal: "127",
			want:   func() *Config { return &Config{Int8Value: 127} },
		},
		{
			name:   "int16 type",
			envVar: "INT16_VALUE",
			envVal: "32767",
			want:   func() *Config { return &Config{Int16Value: 32767} },
		},
		{
			name:   "int32 type",
			envVar: "INT32_VALUE",
			envVal: "2147483647",
			want:   func() *Config { return &Config{Int32Value: 2147483647} },
		},
		{
			name:   "int64 type",
			envVar: "INT64_VALUE",
			envVal: "9223372036854775807",
			want:   func() *Config { return &Config{Int64Value: 9223372036854775807} },
		},
		// unsigned integers
		{
			name:   "uint type",
			envVar: "UINT_VALUE",
			envVal: "100",
			want:   func() *Config { return &Config{UintValue: 100} },
		},
		{
			name:   "uint8 type",
			envVar: "UINT8_VALUE",
			envVal: "255",
			want:   func() *Config { return &Config{Uint8Value: 255} },
		},
		{
			name:   "uint16 type",
			envVar: "UINT16_VALUE",
			envVal: "65535",
			want:   func() *Config { return &Config{Uint16Value: 65535} },
		},
		{
			name:   "uint32 type",
			envVar: "UINT32_VALUE",
			envVal: "4294967295",
			want:   func() *Config { return &Config{Uint32Value: 4294967295} },
		},
		{
			name:   "uint64 type",
			envVar: "UINT64_VALUE",
			envVal: "18446744073709551615",
			want:   func() *Config { return &Config{Uint64Value: 18446744073709551615} },
		},
		// floats
		{
			name:   "float32 type",
			envVar: "FLOAT32_VALUE",
			envVal: "3.14159",
			want:   func() *Config { return &Config{Float32Value: 3.14159} },
		},
		{
			name:   "float64 type",
			envVar: "FLOAT64_VALUE",
			envVal: "3.141592653589793",
			want:   func() *Config { return &Config{Float64Value: 3.141592653589793} },
		},
		// bool
		{
			name:   "bool true",
			envVar: "BOOL_VALUE",
			envVal: "true",
			want:   func() *Config { return &Config{BoolValue: true} },
		},
		{
			name:   "bool false",
			envVar: "BOOL_VALUE",
			envVal: "false",
			want:   func() *Config { return &Config{BoolValue: false} },
		},
		{
			name:   "bool 1",
			envVar: "BOOL_VALUE",
			envVal: "1",
			want:   func() *Config { return &Config{BoolValue: true} },
		},
		{
			name:   "bool 0",
			envVar: "BOOL_VALUE",
			envVal: "0",
			want:   func() *Config { return &Config{BoolValue: false} },
		},
		// string
		{
			name:   "string type",
			envVar: "STRING_VALUE",
			envVal: "hello world",
			want:   func() *Config { return &Config{StringValue: "hello world"} },
		},
		// slices
		{
			name:   "string slice comma separated",
			envVar: "STRING_SLICE",
			envVal: "val1,val2,val3",
			want:   func() *Config { return &Config{StringSlice: []string{"val1", "val2", "val3"}} },
		},
		{
			name:   "string slice semicolon separated",
			envVar: "STRING_SLICE",
			envVal: "val1;val2;val3",
			want:   func() *Config { return &Config{StringSlice: []string{"val1", "val2", "val3"}} },
		},
		{
			name:   "int slice comma separated",
			envVar: "INT_SLICE",
			envVal: "1,2,3,4,5",
			want:   func() *Config { return &Config{IntSlice: []int{1, 2, 3, 4, 5}} },
		},
		{
			name:   "int slice semicolon separated",
			envVar: "INT_SLICE",
			envVal: "1;2;3;4;5",
			want:   func() *Config { return &Config{IntSlice: []int{1, 2, 3, 4, 5}} },
		},
		// maps
		{
			name:   "string map",
			envVar: "MAP_STRING",
			envVal: "key1=val1,key2=val2",
			want: func() *Config {
				return &Config{MapStringValue: map[string]string{"key1": "val1", "key2": "val2"}}
			},
		},
		{
			name:   "int map",
			envVar: "MAP_INT",
			envVal: "key1=1,key2=2",
			want: func() *Config {
				return &Config{MapIntValue: map[string]int{"key1": 1, "key2": 2}}
			},
		},
		// time && duration
		{
			name:   "duration type",
			envVar: "DURATION_VALUE",
			envVal: "15m30s",
			want:   func() *Config { return &Config{DurationValue: 15*time.Minute + 30*time.Second} },
		},
		{
			name:   "time type RFC3339",
			envVar: "TIME_VALUE",
			envVal: "2025-08-28T15:04:05Z",
			want: func() *Config {
				return &Config{TimeValue: time.Date(2025, 8, 28, 15, 4, 5, 0, time.UTC)}
			},
		},
		{
			name:   "time type RFC3339",
			envVar: "TIME_PTR_VALUE",
			envVal: "2025-08-28T15:04:05Z",
			want: func() *Config {
				timeValue := time.Date(2025, 8, 28, 15, 4, 5, 0, time.UTC)
				return &Config{TimePtrValue: &timeValue}
			},
		},
		// arrays
		{
			name:   "string array comma separated",
			envVar: "STRING_ARRAY",
			envVal: "a,b,c",
			want:   func() *Config { return &Config{StringArray: [3]string{"a", "b", "c"}} },
		},
		{
			name:   "string array semicolon separated",
			envVar: "STRING_ARRAY",
			envVal: "a;b;c",
			want:   func() *Config { return &Config{StringArray: [3]string{"a", "b", "c"}} },
		},
		{
			name:   "int array comma separated",
			envVar: "INT_ARRAY",
			envVal: "1,2,3,4,5",
			want:   func() *Config { return &Config{IntArray: [5]int{1, 2, 3, 4, 5}} },
		},
		{
			name:   "int array semicolon separated",
			envVar: "INT_ARRAY",
			envVal: "1;2;3;4;5",
			want:   func() *Config { return &Config{IntArray: [5]int{1, 2, 3, 4, 5}} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.envVal)
			cfgData := &Config{}
			cfg := NewConfig(config.Struct(cfgData))

			require.NoError(t, cfg.Init())
			require.NoError(t, cfg.Load(context.Background()))

			require.Equal(t, tt.want(), cfgData)
		})
	}
}

func TestEnv_TimeType_Override(t *testing.T) {
	type Config struct {
		TimeValue time.Time `env:"TIME"`
	}

	tests := []struct {
		name   string
		cfg    *Config
		envVar string
		envVal string
		want   *Config
	}{
		{
			name:   "init value is empty",
			cfg:    &Config{},
			envVar: "TIME",
			envVal: "2025-08-28T15:04:05Z",
			want: &Config{
				TimeValue: time.Date(2025, 8, 28, 15, 4, 5, 0, time.UTC),
			},
		},
		{
			name: "init value is not empty",
			cfg: &Config{
				TimeValue: time.Date(2025, 5, 25, 15, 5, 5, 5, time.UTC),
			},
			envVar: "TIME",
			envVal: "2025-08-28T15:04:05Z",
			want: &Config{
				TimeValue: time.Date(2025, 8, 28, 15, 4, 5, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.envVal)

			cfg := NewConfig(config.Struct(tt.cfg))
			require.NoError(t, cfg.Init())
			require.NoError(t, cfg.Load(context.Background(), config.LoadOverride(true)))
			require.Equal(t, tt.want, tt.cfg)
		})
	}
}

func TestEnv_TimePointerType_Override(t *testing.T) {
	type Config struct {
		TimeValue *time.Time `env:"TIME"`
	}

	tests := []struct {
		name   string
		cfg    func() *Config
		envVar string
		envVal string
		want   func() *Config
	}{
		{
			name:   "init value is empty",
			cfg:    func() *Config { return &Config{} },
			envVar: "TIME",
			envVal: "2025-08-28T15:04:05Z",
			want: func() *Config {
				timeValue := time.Date(2025, 8, 28, 15, 4, 5, 0, time.UTC)
				return &Config{
					TimeValue: &timeValue,
				}
			},
		},
		{
			name: "init value is not empty",
			cfg: func() *Config {
				timeValue := time.Date(2025, 5, 25, 15, 5, 5, 5, time.UTC)
				return &Config{
					TimeValue: &timeValue,
				}
			},
			envVar: "TIME",
			envVal: "2025-08-28T15:04:05Z",
			want: func() *Config {
				timeValue := time.Date(2025, 8, 28, 15, 4, 5, 0, time.UTC)
				return &Config{
					TimeValue: &timeValue,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.envVal)

			cfgData := tt.cfg()
			cfg := NewConfig(config.Struct(cfgData))
			require.NoError(t, cfg.Init())
			require.NoError(t, cfg.Load(context.Background(), config.LoadOverride(true)))
			require.Equal(t, tt.want(), cfgData)
		})
	}
}

func TestEnv_InvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envVal   string
		makeCfg  func() any
	}{
		{
			name:   "bad map",
			envVar: "BAD_MAP",
			envVal: "novalue",
			makeCfg: func() any {
				return &struct {
					MapVal map[string]string `env:"BAD_MAP"`
				}{}
			},
		},
		{
			name:   "bad array",
			envVar: "BAD_ARRAY",
			envVal: "1",
			makeCfg: func() any {
				return &struct {
					ArrayVal [2]int `env:"BAD_ARRAY"`
				}{}
			},
		},
		{
			name:   "bad duration",
			envVar: "BAD_DURATION",
			envVal: "oops",
			makeCfg: func() any {
				return &struct {
					Duration time.Duration `env:"BAD_DURATION"`
				}{}
			},
		},
		{
			name:   "bad time",
			envVar: "BAD_TIME",
			envVal: "not-a-time",
			makeCfg: func() any {
				return &struct {
					TimeVal time.Time `env:"BAD_TIME"`
				}{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.envVal)

			cfgData := tt.makeCfg()
			c := NewConfig(config.Struct(cfgData))
			require.NoError(t, c.Init())
			err := c.Load(context.Background())
			require.Error(t, err)
		})
	}
}

func TestEnv_AllTypes(t *testing.T) {
	type Config struct {
		BoolVal    bool              `env:"BOOL"`
		StringVal  string            `env:"STRING"`
		IntVal     int               `env:"INT"`
		UintVal    uint              `env:"UINT"`
		Float32Val float32           `env:"FLOAT32"`
		Float64Val float64           `env:"FLOAT64"`
		SliceVal   []string          `env:"SLICE"`
		ArrayVal   [2]int            `env:"ARRAY"`
		MapVal     map[string]string `env:"MAP"`
		Duration   time.Duration     `env:"DURATION"`
		TimeVal    time.Time         `env:"TIME"`
		TimePtrVal *time.Time        `env:"TIME_PTR"`
	}

	now := time.Now().Truncate(time.Second)
	vars := map[string]string{
		"BOOL":     "true",
		"STRING":   "hello",
		"INT":      "42",
		"UINT":     "7",
		"FLOAT32":  "3.14",
		"FLOAT64":  "2.71828",
		"SLICE":    "a,b,c",
		"ARRAY":    "1;2",
		"MAP":      "k1=v1,k2=v2",
		"DURATION": "5s",
		"TIME":     now.Format(time.RFC3339),
		"TIME_PTR": now.Format(time.RFC3339),
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}

	cfg := &Config{}
	c := NewConfig(config.Struct(cfg))
	require.NoError(t, c.Load(context.Background()))

	require.True(t, cfg.BoolVal)
	require.Equal(t, "hello", cfg.StringVal)
	require.Equal(t, 42, cfg.IntVal)
	require.Equal(t, uint(7), cfg.UintVal)
	require.InEpsilon(t, 3.14, cfg.Float32Val, 1e-6)
	require.InEpsilon(t, 2.71828, cfg.Float64Val, 1e-6)
	require.Equal(t, []string{"a", "b", "c"}, cfg.SliceVal)
	require.Equal(t, [2]int{1, 2}, cfg.ArrayVal)
	require.Equal(t, map[string]string{"k1": "v1", "k2": "v2"}, cfg.MapVal)
	require.Equal(t, 5*time.Second, cfg.Duration)
	require.Equal(t, now, cfg.TimeVal)
	require.NotNil(t, cfg.TimePtrVal)
	require.Equal(t, now, *cfg.TimePtrVal)

	// Roundtrip Save to Load
	require.NoError(t, c.Save(context.Background()))
	for k := range vars {
		val, ok := os.LookupEnv(k)
		require.True(t, ok, "env var %s must exist", k)
		require.NotEmpty(t, val)
	}
}
