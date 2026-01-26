package ldap

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
)

var (
	ErrPoolClosed    = errors.New("connection pool is closed")
	ErrPoolExhausted = errors.New("connection pool exhausted")
)

// PoolConfig represents connection pool configuration
type PoolConfig struct {
	MaxConns    int           // Maximum number of connections in the pool
	MinConns    int           // Minimum number of idle connections to maintain
	MaxIdleTime time.Duration // Maximum time a connection can be idle before being closed
	DialTimeout time.Duration // Timeout for establishing new connections
	HealthCheck time.Duration // Interval for health check pings
}

// DefaultPoolConfig returns sensible defaults for the connection pool
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:    10,
		MinConns:    2,
		MaxIdleTime: 10 * time.Minute,
		DialTimeout: 10 * time.Second,
		HealthCheck: 30 * time.Second,
	}
}

// pooledConn wraps an LDAP connection with metadata
type pooledConn struct {
	conn      *ldap.Conn
	lastUsed  time.Time
	inUse     bool
	unhealthy bool
}

// ConnectionPool manages a pool of LDAP connections
type ConnectionPool struct {
	config     *Config
	poolConfig PoolConfig
	conns      []*pooledConn
	mu         sync.Mutex
	closed     bool
	cond       *sync.Cond
}

// NewConnectionPool creates a new LDAP connection pool
func NewConnectionPool(config *Config, poolConfig PoolConfig) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		config:     config,
		poolConfig: poolConfig,
		conns:      make([]*pooledConn, 0, poolConfig.MaxConns),
		closed:     false,
	}
	pool.cond = sync.NewCond(&pool.mu)

	// Pre-create minimum connections
	for i := 0; i < poolConfig.MinConns; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			// Clean up any connections we've created
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection %d: %w", i+1, err)
		}
		pool.conns = append(pool.conns, &pooledConn{
			conn:     conn,
			lastUsed: time.Now(),
			inUse:    false,
		})
	}

	// Start background maintenance goroutine
	go pool.maintain()

	return pool, nil
}

// createConnection creates a new LDAP connection
func (p *ConnectionPool) createConnection() (*ldap.Conn, error) {
	// Parse timeout
	timeout := p.poolConfig.DialTimeout
	if p.config.Timeout != "" {
		if t, err := time.ParseDuration(p.config.Timeout); err == nil {
			timeout = t
		}
	}

	// Build LDAP URL
	protocol := "ldap"
	if p.config.UseTLS {
		protocol = "ldaps"
	}
	url := fmt.Sprintf("%s://%s", protocol, p.config.Server)

	// Connect with timeout using a dialer
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := ldap.DialURL(url, ldap.DialWithDialer(dialer))
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP server: %w", err)
	}

	// Bind with service account
	if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind: %w", err)
	}

	return conn, nil
}

// Acquire gets a connection from the pool
// Blocks until a connection is available or context is cancelled
func (p *ConnectionPool) Acquire() (*ldap.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	// Wait for an available connection
	for {
		// Look for an idle, healthy connection
		for _, pc := range p.conns {
			if !pc.inUse && !pc.unhealthy {
				pc.inUse = true
				pc.lastUsed = time.Now()
				return pc.conn, nil
			}
		}

		// No idle connections available - try to create a new one
		if len(p.conns) < p.poolConfig.MaxConns {
			conn, err := p.createConnection()
			if err != nil {
				return nil, fmt.Errorf("failed to create new connection: %w", err)
			}

			pc := &pooledConn{
				conn:     conn,
				lastUsed: time.Now(),
				inUse:    true,
			}
			p.conns = append(p.conns, pc)
			return conn, nil
		}

		// Pool is exhausted - wait for a connection to be released
		p.cond.Wait()

		if p.closed {
			return nil, ErrPoolClosed
		}
	}
}

// Release returns a connection to the pool
func (p *ConnectionPool) Release(conn *ldap.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find the pooled connection
	for _, pc := range p.conns {
		if pc.conn == conn {
			pc.inUse = false
			pc.lastUsed = time.Now()
			p.cond.Signal() // Wake up one waiting goroutine
			return
		}
	}
}

// MarkUnhealthy marks a connection as unhealthy
// Should be called when an operation fails
func (p *ConnectionPool) MarkUnhealthy(conn *ldap.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pc := range p.conns {
		if pc.conn == conn {
			pc.unhealthy = true
			pc.inUse = false
			p.cond.Signal() // Wake up waiting goroutines
			return
		}
	}
}

// maintain runs background maintenance tasks
func (p *ConnectionPool) maintain() {
	ticker := time.NewTicker(p.poolConfig.HealthCheck)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return
		}

		now := time.Now()
		newConns := make([]*pooledConn, 0, len(p.conns))

		for _, pc := range p.conns {
			// Remove unhealthy connections
			if pc.unhealthy && !pc.inUse {
				pc.conn.Close()
				continue
			}

			// Remove idle connections that exceed max idle time
			// but maintain minimum pool size
			idleTime := now.Sub(pc.lastUsed)
			if !pc.inUse && idleTime > p.poolConfig.MaxIdleTime && len(newConns) >= p.poolConfig.MinConns {
				pc.conn.Close()
				continue
			}

			// Health check idle connections
			if !pc.inUse && !pc.unhealthy {
				// Simple health check: try a bind operation
				if err := pc.conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
					pc.unhealthy = true
					pc.conn.Close()
					continue
				}
			}

			newConns = append(newConns, pc)
		}

		p.conns = newConns

		// Ensure we maintain minimum connections
		for len(p.conns) < p.poolConfig.MinConns {
			conn, err := p.createConnection()
			if err != nil {
				// Log error but continue - we'll try again next cycle
				break
			}
			p.conns = append(p.conns, &pooledConn{
				conn:     conn,
				lastUsed: time.Now(),
				inUse:    false,
			})
		}

		p.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.cond.Broadcast() // Wake up all waiting goroutines

	for _, pc := range p.conns {
		if pc.conn != nil {
			pc.conn.Close()
		}
	}

	p.conns = nil
	return nil
}

// Stats returns current pool statistics
type PoolStats struct {
	TotalConns     int
	IdleConns      int
	ActiveConns    int
	UnhealthyConns int
}

// Stats returns current pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats := PoolStats{
		TotalConns: len(p.conns),
	}

	for _, pc := range p.conns {
		if pc.unhealthy {
			stats.UnhealthyConns++
		} else if pc.inUse {
			stats.ActiveConns++
		} else {
			stats.IdleConns++
		}
	}

	return stats
}
