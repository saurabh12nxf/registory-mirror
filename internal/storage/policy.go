package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type CachePolicy struct {
	ID         int
	Image      string
	TTL        int64 // seconds
	ExpiresAt  time.Time
	CreatedAt  time.Time
	CreatedBy  string
	Reason     string
}

type PolicyAuditLog struct {
	ID        int
	Image     string
	Action    string // "created", "expired", "revoked"
	TTL       int64
	ExpiresAt time.Time
	Reason    string
	Timestamp time.Time
}

// InitPolicySchema creates the policy and audit tables
func (db *DB) InitPolicySchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS cache_policies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		image TEXT NOT NULL UNIQUE,
		ttl INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT DEFAULT 'system',
		reason TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_policy_image ON cache_policies(image);
	CREATE INDEX IF NOT EXISTS idx_policy_expires ON cache_policies(expires_at);

	CREATE TABLE IF NOT EXISTS policy_audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		image TEXT NOT NULL,
		action TEXT NOT NULL,
		ttl INTEGER,
		expires_at DATETIME,
		reason TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_audit_image ON policy_audit_log(image);
	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON policy_audit_log(timestamp);
	`
	_, err := db.conn.Exec(query)
	return err
}

// SetCachePolicy sets a TTL-based policy for an image
func (db *DB) SetCachePolicy(image string, ttlSeconds int64, reason string) error {
	expiresAt := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	
	// Insert or replace policy
	query := `INSERT OR REPLACE INTO cache_policies (image, ttl, expires_at, reason) VALUES (?, ?, ?, ?)`
	_, err := db.conn.Exec(query, image, ttlSeconds, expiresAt, reason)
	if err != nil {
		return err
	}

	// Log the action
	return db.logPolicyAction(image, "created", ttlSeconds, expiresAt, reason)
}

// GetCachePolicy retrieves the policy for an image
func (db *DB) GetCachePolicy(image string) (*CachePolicy, error) {
	query := `SELECT id, image, ttl, expires_at, created_at, created_by, reason 
	          FROM cache_policies WHERE image = ?`
	
	row := db.conn.QueryRow(query, image)
	var policy CachePolicy
	err := row.Scan(&policy.ID, &policy.Image, &policy.TTL, &policy.ExpiresAt, 
	               &policy.CreatedAt, &policy.CreatedBy, &policy.Reason)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &policy, nil
}

// IsImageExpired checks if an image's cache has expired
func (db *DB) IsImageExpired(image string) (bool, error) {
	policy, err := db.GetCachePolicy(image)
	if err != nil {
		return false, err
	}
	
	if policy == nil {
		return false, nil // No policy means no expiration
	}
	
	return time.Now().After(policy.ExpiresAt), nil
}

// CleanupExpiredPolicies removes expired policies and logs them
func (db *DB) CleanupExpiredPolicies() (int, error) {
	// Get expired policies
	query := `SELECT image, ttl, expires_at, reason FROM cache_policies WHERE expires_at < ?`
	rows, err := db.conn.Query(query, time.Now())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var image, reason string
		var ttl int64
		var expiresAt time.Time
		
		if err := rows.Scan(&image, &ttl, &expiresAt, &reason); err != nil {
			continue
		}
		
		// Log expiration
		db.logPolicyAction(image, "expired", ttl, expiresAt, reason)
		count++
	}

	// Delete expired policies
	deleteQuery := `DELETE FROM cache_policies WHERE expires_at < ?`
	_, err = db.conn.Exec(deleteQuery, time.Now())
	if err != nil {
		return count, err
	}

	return count, nil
}

// GetExpiringPolicies returns policies expiring within the given duration
func (db *DB) GetExpiringPolicies(within time.Duration) ([]CachePolicy, error) {
	expiryThreshold := time.Now().Add(within)
	
	query := `SELECT id, image, ttl, expires_at, created_at, created_by, reason 
	          FROM cache_policies 
	          WHERE expires_at > ? AND expires_at < ?
	          ORDER BY expires_at ASC`
	
	rows, err := db.conn.Query(query, time.Now(), expiryThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []CachePolicy
	for rows.Next() {
		var p CachePolicy
		if err := rows.Scan(&p.ID, &p.Image, &p.TTL, &p.ExpiresAt, &p.CreatedAt, &p.CreatedBy, &p.Reason); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	
	return policies, nil
}

// RevokeCachePolicy manually revokes a policy before expiration
func (db *DB) RevokeCachePolicy(image string, reason string) error {
	// Get existing policy for audit
	policy, err := db.GetCachePolicy(image)
	if err != nil {
		return err
	}
	if policy == nil {
		return fmt.Errorf("no policy found for image: %s", image)
	}

	// Log revocation
	if err := db.logPolicyAction(image, "revoked", policy.TTL, policy.ExpiresAt, reason); err != nil {
		return err
	}

	// Delete policy
	query := `DELETE FROM cache_policies WHERE image = ?`
	_, err = db.conn.Exec(query, image)
	return err
}

// GetPolicyAuditLog retrieves audit log entries
func (db *DB) GetPolicyAuditLog(limit int) ([]PolicyAuditLog, error) {
	query := `SELECT id, image, action, ttl, expires_at, reason, timestamp 
	          FROM policy_audit_log 
	          ORDER BY timestamp DESC 
	          LIMIT ?`
	
	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []PolicyAuditLog
	for rows.Next() {
		var log PolicyAuditLog
		if err := rows.Scan(&log.ID, &log.Image, &log.Action, &log.TTL, &log.ExpiresAt, &log.Reason, &log.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	
	return logs, nil
}

// logPolicyAction logs a policy action to the audit log
func (db *DB) logPolicyAction(image, action string, ttl int64, expiresAt time.Time, reason string) error {
	query := `INSERT INTO policy_audit_log (image, action, ttl, expires_at, reason) VALUES (?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, image, action, ttl, expiresAt, reason)
	return err
}

// GetAllActivePolicies returns all non-expired policies
func (db *DB) GetAllActivePolicies() ([]CachePolicy, error) {
	query := `SELECT id, image, ttl, expires_at, created_at, created_by, reason 
	          FROM cache_policies 
	          WHERE expires_at > ?
	          ORDER BY expires_at ASC`
	
	rows, err := db.conn.Query(query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []CachePolicy
	for rows.Next() {
		var p CachePolicy
		if err := rows.Scan(&p.ID, &p.Image, &p.TTL, &p.ExpiresAt, &p.CreatedAt, &p.CreatedBy, &p.Reason); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	
	return policies, nil
}
