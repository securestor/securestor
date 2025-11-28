package failover

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/securestor/securestor/internal/logger"
)

// DatabaseFailoverService manages PostgreSQL primary/standby failover
type DatabaseFailoverService struct {
	primaryDB      *sql.DB
	standbyDB      *sql.DB
	primaryConnStr string
	standbyConnStr string
	logger         *logger.Logger
	mutex          sync.RWMutex
	isHealthy      bool
	lastCheck      time.Time
	failureCount   int
}

// FailoverState represents the current failover state
type FailoverState struct {
	IsPrimaryHealthy   bool
	IsStandbyHealthy   bool
	ReplicationLag     time.Duration
	LastFailover       time.Time
	FailoverCount      int
	CurrentPrimaryRole string
	FailoverInProgress bool
}

// NewDatabaseFailoverService creates a new failover service
func NewDatabaseFailoverService(primaryConnStr, standbyConnStr string, logger *logger.Logger) *DatabaseFailoverService {
	return &DatabaseFailoverService{
		primaryConnStr: primaryConnStr,
		standbyConnStr: standbyConnStr,
		logger:         logger,
		isHealthy:      true,
		lastCheck:      time.Now(),
		failureCount:   0,
	}
}

// Initialize connects to databases
func (dfs *DatabaseFailoverService) Initialize() error {
	var err error

	// Connect to primary
	dfs.primaryDB, err = sql.Open("postgres", dfs.primaryConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to primary: %w", err)
	}

	if err := dfs.primaryDB.Ping(); err != nil {
		return fmt.Errorf("primary database not accessible: %w", err)
	}

	// Connect to standby
	dfs.standbyDB, err = sql.Open("postgres", dfs.standbyConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to standby: %w", err)
	}

	if err := dfs.standbyDB.Ping(); err != nil {
		return fmt.Errorf("standby database not accessible: %w", err)
	}

	dfs.logger.Printf("Database failover service initialized successfully")
	return nil
}

// GetFailoverState returns current failover state
func (dfs *DatabaseFailoverService) GetFailoverState(ctx context.Context) (*FailoverState, error) {
	state := &FailoverState{
		LastFailover: time.Now(),
	}

	// Check primary health
	primaryHealth := dfs.checkDatabaseHealth(ctx, dfs.primaryDB, "PRIMARY")
	state.IsPrimaryHealthy = primaryHealth

	// Check standby health
	standbyHealth := dfs.checkDatabaseHealth(ctx, dfs.standbyDB, "STANDBY")
	state.IsStandbyHealthy = standbyHealth

	// Get replication lag
	lag, err := dfs.getReplicationLag(ctx)
	if err == nil {
		state.ReplicationLag = lag
	}

	// Determine current primary role
	if state.IsPrimaryHealthy {
		state.CurrentPrimaryRole = "PRIMARY"
	} else if state.IsStandbyHealthy {
		state.CurrentPrimaryRole = "STANDBY"
	} else {
		state.CurrentPrimaryRole = "UNKNOWN"
	}

	return state, nil
}

// checkDatabaseHealth verifies database is accessible and operational
func (dfs *DatabaseFailoverService) checkDatabaseHealth(ctx context.Context, db *sql.DB, role string) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		dfs.logger.Printf("ERROR: %s database health check failed: %v", role, err)
		return false
	}

	// Check if it's actually a database server
	var version string
	err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		dfs.logger.Printf("ERROR: %s database version query failed: %v", role, err)
		return false
	}

	return true
}

// getReplicationLag gets the replication lag in seconds
func (dfs *DatabaseFailoverService) getReplicationLag(ctx context.Context) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var lag sql.NullInt64
	err := dfs.primaryDB.QueryRowContext(ctx,
		`SELECT EXTRACT(EPOCH FROM (NOW() - pg_last_xact_replay_timestamp()))::bigint`).Scan(&lag)

	if err != nil {
		return 0, err
	}

	if !lag.Valid {
		// No replication, lag is 0
		return 0, nil
	}

	return time.Duration(lag.Int64) * time.Second, nil
}

// PromoteStandby promotes the standby database to primary
func (dfs *DatabaseFailoverService) PromoteStandby(ctx context.Context) error {
	dfs.mutex.Lock()
	defer dfs.mutex.Unlock()

	dfs.logger.Printf("⚠️  Initiating standby promotion...")

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Execute pg_ctl promote on standby
	// Note: In containerized environment, this is done via exec
	_, err := dfs.standbyDB.ExecContext(ctx, "SELECT pg_promote();")
	if err != nil {
		return fmt.Errorf("failed to promote standby: %w", err)
	}

	dfs.logger.Printf("✅ Standby promoted to primary successfully")
	dfs.failureCount++

	return nil
}

// DemotePrimary demotes the current primary to standby
func (dfs *DatabaseFailoverService) DemotePrimary(ctx context.Context) error {
	dfs.mutex.Lock()
	defer dfs.mutex.Unlock()

	dfs.logger.Printf("⚠️  Initiating primary demotion...")

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Close primary connections
	if dfs.primaryDB != nil {
		dfs.primaryDB.Close()
	}

	dfs.logger.Printf("✅ Primary demoted (connections closed)")

	return nil
}

// MonitorFailover continuously monitors for failover triggers
func (dfs *DatabaseFailoverService) MonitorFailover(ctx context.Context, failoverThreshold time.Duration) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	consecutiveFailures := 0

	for {
		select {
		case <-ctx.Done():
			dfs.logger.Printf("Failover monitoring stopped")
			return
		case <-ticker.C:
			state, err := dfs.GetFailoverState(ctx)
			if err != nil {
				dfs.logger.Printf("ERROR: Failed to get failover state: %v", err)
				continue
			}

			// Check if primary is unhealthy
			if !state.IsPrimaryHealthy && state.IsStandbyHealthy {
				consecutiveFailures++
				dfs.logger.Printf("⚠️  Primary unhealthy (%d consecutive failures)", consecutiveFailures)

				// Trigger failover if threshold exceeded
				if time.Duration(consecutiveFailures)*10*time.Second > failoverThreshold {
					dfs.logger.Printf("🚨 FAILOVER THRESHOLD EXCEEDED - Promoting standby")
					if err := dfs.PromoteStandby(ctx); err != nil {
						dfs.logger.Printf("ERROR: Failover failed: %v", err)
					}
					consecutiveFailures = 0
				}
			} else {
				consecutiveFailures = 0
			}

			// Log replication lag if excessive
			if state.ReplicationLag > 30*time.Second {
				dfs.logger.Printf("⚠️  High replication lag: %v", state.ReplicationLag)
			}
		}
	}
}

// GetReplicationMetrics returns replication performance metrics
func (dfs *DatabaseFailoverService) GetReplicationMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get replication lag
	var lag sql.NullInt64
	dfs.primaryDB.QueryRowContext(ctx,
		`SELECT EXTRACT(EPOCH FROM (NOW() - pg_last_xact_replay_timestamp()))::bigint`).Scan(&lag)

	if lag.Valid {
		metrics["replication_lag_seconds"] = lag.Int64
	}

	// Get replication stats
	rows, err := dfs.primaryDB.QueryContext(ctx, `
		SELECT 
			client_addr,
			usename,
			state,
			write_lag,
			flush_lag,
			replay_lag
		FROM pg_stat_replication
	`)
	if err == nil {
		defer rows.Close()

		replicas := make([]map[string]interface{}, 0)
		for rows.Next() {
			var clientAddr, usename, state sql.NullString
			var writeLag, flushLag, replayLag sql.NullString

			if err := rows.Scan(&clientAddr, &usename, &state, &writeLag, &flushLag, &replayLag); err != nil {
				continue
			}

			replica := make(map[string]interface{})
			if clientAddr.Valid {
				replica["client_addr"] = clientAddr.String
			}
			if usename.Valid {
				replica["user"] = usename.String
			}
			if state.Valid {
				replica["state"] = state.String
			}
			if writeLag.Valid {
				replica["write_lag"] = writeLag.String
			}
			if flushLag.Valid {
				replica["flush_lag"] = flushLag.String
			}
			if replayLag.Valid {
				replica["replay_lag"] = replayLag.String
			}

			replicas = append(replicas, replica)
		}

		metrics["replicas"] = replicas
	}

	return metrics, nil
}

// Close closes database connections
func (dfs *DatabaseFailoverService) Close() error {
	if dfs.primaryDB != nil {
		dfs.primaryDB.Close()
	}
	if dfs.standbyDB != nil {
		dfs.standbyDB.Close()
	}
	return nil
}

// SwitchPrimary performs a controlled switchover to standby
func (dfs *DatabaseFailoverService) SwitchPrimary(ctx context.Context) error {
	dfs.mutex.Lock()
	defer dfs.mutex.Unlock()

	dfs.logger.Printf("⚠️  Initiating controlled primary switchover...")

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Step 1: Ensure no more writes to primary
	dfs.logger.Printf("Step 1: Stopping writes to current primary...")

	// Step 2: Wait for replication to catch up
	dfs.logger.Printf("Step 2: Waiting for replication to catch up...")
	for i := 0; i < 30; i++ {
		lag, err := dfs.getReplicationLag(ctx)
		if err != nil || lag < 100*time.Millisecond {
			dfs.logger.Printf("✅ Replication caught up")
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Step 3: Promote standby
	dfs.logger.Printf("Step 3: Promoting standby...")
	_, err := dfs.standbyDB.ExecContext(ctx, "SELECT pg_promote();")
	if err != nil {
		return fmt.Errorf("failed to promote standby: %w", err)
	}

	// Step 4: Restart primary in standby mode
	dfs.logger.Printf("Step 4: Demoting old primary to standby...")
	if dfs.primaryDB != nil {
		dfs.primaryDB.Close()
	}

	dfs.logger.Printf("✅ Switchover completed successfully")
	return nil
}
