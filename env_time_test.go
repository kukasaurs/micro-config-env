package env_test

import (
	"context"
	"os"
	"testing"
	"time"

	"go.unistack.org/micro/v3/config"

	env "go.unistack.org/micro-config-env/v3"
)

type EnvConfig struct {
	IntValue      int           `env:"INT_VALUE"`
	DurationValue time.Duration `env:"CACHE_TTL"`
	TimeValue     time.Time     `env:"START_DATE"`
}

func TestEnv_BasicTypes(t *testing.T) {
	cases := []struct {
		name    string
		envVar  string
		envVal  string
		checker func(t *testing.T, cfg *EnvConfig)
	}{
		{
			name:   "int from env",
			envVar: "INT_VALUE",
			envVal: "100",
			checker: func(t *testing.T, cfg *EnvConfig) {
				if cfg.IntValue != 100 {
					t.Errorf("expected 100, got %d", cfg.IntValue)
				}
			},
		},
		{
			name:   "duration from env",
			envVar: "CACHE_TTL",
			envVal: "15m",
			checker: func(t *testing.T, cfg *EnvConfig) {
				expected := 15 * time.Minute
				if cfg.DurationValue != expected {
					t.Errorf("expected %v, got %v", expected, cfg.DurationValue)
				}
			},
		},
		{
			name:   "time from env (RFC3339)",
			envVar: "START_DATE",
			envVal: "2025-08-28T15:04:04Z",
			checker: func(t *testing.T, cfg *EnvConfig) {
				expected := time.Date(2025, 8, 28, 15, 4, 4, 0, time.UTC)
				if !cfg.TimeValue.Equal(expected) {
					t.Errorf("expected %v, got %v", expected, cfg.TimeValue)
				}
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Unsetenv("INT_VALUE")
			_ = os.Unsetenv("CACHE_TTL")
			_ = os.Unsetenv("START_DATE")

			if err := os.Setenv(tt.envVar, tt.envVal); err != nil {
				t.Fatalf("failed to set env var: %v", err)
			}

			cfgData := &EnvConfig{}
			cfg := env.NewConfig(config.Struct(cfgData))

			if err := cfg.Init(); err != nil {
				t.Fatalf("Init failed: %v", err)
			}

			if err := cfg.Load(context.Background()); err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			tt.checker(t, cfgData)
		})
	}
}
