package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_PoolWarning(t *testing.T) {
	tests := []struct {
		name        string
		maxConns    int
		concurrency int
		wantWarning bool
	}{
		{"pool larger than concurrency", 60, 50, false},
		{"pool equal to concurrency", 50, 50, false},
		{"pool smaller than concurrency", 25, 50, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DB:     DBConfig{MaxConns: tt.maxConns},
				Worker: WorkerConfig{Concurrency: tt.concurrency},
			}
			warning := cfg.PoolWarning()
			if tt.wantWarning {
				assert.NotEmpty(t, warning)
				// Le message doit citer les deux valeurs pour être actionnable.
				assert.True(t, strings.Contains(warning, "25"), "warning should mention pool size")
				assert.True(t, strings.Contains(warning, "50"), "warning should mention concurrency")
			} else {
				assert.Empty(t, warning)
			}
		})
	}
}
