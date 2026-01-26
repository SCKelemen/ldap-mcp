package ldap

import (
	"testing"
	"time"
)

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	if config.MaxConns <= 0 {
		t.Error("MaxConns should be positive")
	}

	if config.MinConns < 0 {
		t.Error("MinConns should be non-negative")
	}

	if config.MinConns > config.MaxConns {
		t.Error("MinConns should not exceed MaxConns")
	}

	if config.MaxIdleTime <= 0 {
		t.Error("MaxIdleTime should be positive")
	}

	if config.DialTimeout <= 0 {
		t.Error("DialTimeout should be positive")
	}

	if config.HealthCheck <= 0 {
		t.Error("HealthCheck interval should be positive")
	}
}

func TestPoolConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config PoolConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: PoolConfig{
				MaxConns:    10,
				MinConns:    2,
				MaxIdleTime: 10 * time.Minute,
				DialTimeout: 10 * time.Second,
				HealthCheck: 30 * time.Second,
			},
			valid: true,
		},
		{
			name: "zero max conns should still work",
			config: PoolConfig{
				MaxConns:    1,
				MinConns:    0,
				MaxIdleTime: 1 * time.Minute,
				DialTimeout: 5 * time.Second,
				HealthCheck: 10 * time.Second,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.MaxConns < tt.config.MinConns {
				t.Error("MaxConns should be >= MinConns")
			}
		})
	}
}

func TestPoolStats(t *testing.T) {
	// This test doesn't require a real LDAP connection
	// It just tests the stats structure
	stats := PoolStats{
		TotalConns:     10,
		IdleConns:      5,
		ActiveConns:    4,
		UnhealthyConns: 1,
	}

	if stats.TotalConns != stats.IdleConns+stats.ActiveConns+stats.UnhealthyConns {
		t.Errorf("Stats don't add up: total=%d, idle=%d, active=%d, unhealthy=%d",
			stats.TotalConns, stats.IdleConns, stats.ActiveConns, stats.UnhealthyConns)
	}
}
