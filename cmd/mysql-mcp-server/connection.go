// cmd/mysql-mcp-server/connection.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/askdba/mysql-mcp-server/internal/util"
)

// ===== Multi-DSN Connection Manager =====

// ConnectionConfig represents a single MySQL connection configuration.
type ConnectionConfig struct {
	Name        string `json:"name"`
	DSN         string `json:"dsn"`
	Description string `json:"description,omitempty"`
	ReadOnly    bool   `json:"read_only,omitempty"`
}

// ConnectionManager manages multiple MySQL connections.
type ConnectionManager struct {
	connections map[string]*sql.DB
	configs     map[string]ConnectionConfig
	activeConn  string
	mu          sync.RWMutex
}

// NewConnectionManager creates a new connection manager.
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*sql.DB),
		configs:     make(map[string]ConnectionConfig),
	}
}

// AddConnection adds a new connection to the manager.
func (cm *ConnectionManager) AddConnection(cfg ConnectionConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	conn, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return fmt.Errorf("failed to open connection %s: %w", cfg.Name, err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(defaultMaxOpenConns)
	conn.SetMaxIdleConns(defaultMaxIdleConns)
	conn.SetConnMaxLifetime(time.Duration(defaultConnMaxLifetimeM) * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("failed to ping connection %s: %w", cfg.Name, err)
	}

	cm.connections[cfg.Name] = conn
	cm.configs[cfg.Name] = cfg

	// Set as active if it's the first connection
	if cm.activeConn == "" {
		cm.activeConn = cfg.Name
	}

	return nil
}

// GetActive returns the active database connection and its name.
func (cm *ConnectionManager) GetActive() (*sql.DB, string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connections[cm.activeConn], cm.activeConn
}

// SetActive sets the active connection by name.
func (cm *ConnectionManager) SetActive(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.connections[name]; !exists {
		return fmt.Errorf("connection '%s' not found", name)
	}
	cm.activeConn = name
	return nil
}

// List returns a list of all connection configurations with masked DSNs.
func (cm *ConnectionManager) List() []ConnectionConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var list []ConnectionConfig
	for _, cfg := range cm.configs {
		// Mask DSN for security
		maskedCfg := cfg
		maskedCfg.DSN = util.MaskDSN(cfg.DSN)
		list = append(list, maskedCfg)
	}
	return list
}

// GetActiveDB returns the active database connection.
func (cm *ConnectionManager) GetActiveDB() *sql.DB {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connections[cm.activeConn]
}

// Close closes all connections managed by the manager.
func (cm *ConnectionManager) Close() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for _, conn := range cm.connections {
		conn.Close()
	}
}

// getDB returns the active database connection in a thread-safe manner.
// This should be used instead of accessing the global db variable directly
// to avoid data races when connections are switched.
func getDB() *sql.DB {
	if connManager != nil {
		return connManager.GetActiveDB()
	}
	return db
}

// loadConnectionsFromEnv loads DSN configurations from environment variables.
func loadConnectionsFromEnv() ([]ConnectionConfig, error) {
	var configs []ConnectionConfig

	// Check for JSON-based configuration first
	if jsonConfig := os.Getenv("MYSQL_CONNECTIONS"); jsonConfig != "" {
		if err := json.Unmarshal([]byte(jsonConfig), &configs); err != nil {
			return nil, fmt.Errorf("failed to parse MYSQL_CONNECTIONS: %w", err)
		}
		return configs, nil
	}

	// Fall back to numbered DSN environment variables
	// MYSQL_DSN (default), MYSQL_DSN_1, MYSQL_DSN_2, etc.
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		configs = append(configs, ConnectionConfig{
			Name:        "default",
			DSN:         dsn,
			Description: "Default connection",
		})
	}

	for i := 1; i <= 10; i++ {
		dsnKey := fmt.Sprintf("MYSQL_DSN_%d", i)
		nameKey := fmt.Sprintf("MYSQL_DSN_%d_NAME", i)
		descKey := fmt.Sprintf("MYSQL_DSN_%d_DESC", i)

		dsn := os.Getenv(dsnKey)
		if dsn == "" {
			continue
		}

		name := os.Getenv(nameKey)
		if name == "" {
			name = fmt.Sprintf("connection_%d", i)
		}

		configs = append(configs, ConnectionConfig{
			Name:        name,
			DSN:         dsn,
			Description: os.Getenv(descKey),
		})
	}

	return configs, nil
}

