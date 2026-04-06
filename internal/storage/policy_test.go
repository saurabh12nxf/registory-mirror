package storage

import (
	"os"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "registry-mirror-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Save original HOME and set to temp
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows fallback
	}

	// Set both HOME and USERPROFILE for cross-platform support
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)

	db, err := NewDB()
	if err != nil {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalHome)
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalHome)
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestSetCachePolicy(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name       string
		image      string
		ttlSeconds int64
		reason     string
		wantErr    bool
	}{
		{
			name:       "valid policy",
			image:      "nginx:latest",
			ttlSeconds: 3600,
			reason:     "test policy",
			wantErr:    false,
		},
		{
			name:       "policy with long TTL",
			image:      "redis:latest",
			ttlSeconds: 86400,
			reason:     "24 hour cache",
			wantErr:    false,
		},
		{
			name:       "policy without reason",
			image:      "alpine:latest",
			ttlSeconds: 7200,
			reason:     "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.SetCachePolicy(tt.image, tt.ttlSeconds, tt.reason)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetCachePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify policy was set
				policy, err := db.GetCachePolicy(tt.image)
				if err != nil {
					t.Errorf("GetCachePolicy() error = %v", err)
				}
				if policy == nil {
					t.Error("expected policy to be set, got nil")
				}
				if policy.Image != tt.image {
					t.Errorf("expected image %s, got %s", tt.image, policy.Image)
				}
				if policy.TTL != tt.ttlSeconds {
					t.Errorf("expected TTL %d, got %d", tt.ttlSeconds, policy.TTL)
				}
			}
		})
	}
}

func TestGetCachePolicy(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set up test data
	testImage := "nginx:latest"
	testTTL := int64(3600)
	testReason := "test policy"

	err := db.SetCachePolicy(testImage, testTTL, testReason)
	if err != nil {
		t.Fatalf("failed to set test policy: %v", err)
	}

	tests := []struct {
		name      string
		image     string
		wantNil   bool
		wantImage string
	}{
		{
			name:      "existing policy",
			image:     testImage,
			wantNil:   false,
			wantImage: testImage,
		},
		{
			name:    "non-existent policy",
			image:   "nonexistent:latest",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := db.GetCachePolicy(tt.image)
			if err != nil {
				t.Errorf("GetCachePolicy() error = %v", err)
			}

			if tt.wantNil && policy != nil {
				t.Error("expected nil policy, got non-nil")
			}

			if !tt.wantNil {
				if policy == nil {
					t.Error("expected non-nil policy, got nil")
				} else if policy.Image != tt.wantImage {
					t.Errorf("expected image %s, got %s", tt.wantImage, policy.Image)
				}
			}
		})
	}
}

func TestIsImageExpired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		image       string
		ttlSeconds  int64
		wantExpired bool
	}{
		{
			name:        "not expired",
			image:       "nginx:latest",
			ttlSeconds:  3600,
			wantExpired: false,
		},
		{
			name:        "expired",
			image:       "redis:latest",
			ttlSeconds:  -1, // Already expired
			wantExpired: true,
		},
		{
			name:        "no policy",
			image:       "alpine:latest",
			ttlSeconds:  0,
			wantExpired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ttlSeconds != 0 {
				err := db.SetCachePolicy(tt.image, tt.ttlSeconds, "test")
				if err != nil {
					t.Fatalf("failed to set policy: %v", err)
				}
			}

			expired, err := db.IsImageExpired(tt.image)
			if err != nil {
				t.Errorf("IsImageExpired() error = %v", err)
			}

			if expired != tt.wantExpired {
				t.Errorf("IsImageExpired() = %v, want %v", expired, tt.wantExpired)
			}
		})
	}
}

func TestCleanupExpiredPolicies(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set up test data with expired and non-expired policies
	db.SetCachePolicy("expired1:latest", -1, "expired")
	db.SetCachePolicy("expired2:latest", -1, "expired")
	db.SetCachePolicy("active:latest", 3600, "active")

	count, err := db.CleanupExpiredPolicies()
	if err != nil {
		t.Errorf("CleanupExpiredPolicies() error = %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 expired policies, got %d", count)
	}

	// Verify expired policies are gone
	policy, _ := db.GetCachePolicy("expired1:latest")
	if policy != nil {
		t.Error("expected expired1 policy to be deleted")
	}

	// Verify active policy still exists
	policy, _ = db.GetCachePolicy("active:latest")
	if policy == nil {
		t.Error("expected active policy to still exist")
	}
}

func TestGetExpiringPolicies(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set up test data
	db.SetCachePolicy("soon:latest", 60, "expires soon")     // 1 minute
	db.SetCachePolicy("later:latest", 7200, "expires later") // 2 hours
	db.SetCachePolicy("expired:latest", -1, "already expired")

	policies, err := db.GetExpiringPolicies(5 * time.Minute)
	if err != nil {
		t.Errorf("GetExpiringPolicies() error = %v", err)
	}

	// Should only get the "soon" policy
	if len(policies) != 1 {
		t.Errorf("expected 1 expiring policy, got %d", len(policies))
	}

	if len(policies) > 0 && policies[0].Image != "soon:latest" {
		t.Errorf("expected soon:latest, got %s", policies[0].Image)
	}
}

func TestRevokeCachePolicy(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testImage := "nginx:latest"
	db.SetCachePolicy(testImage, 3600, "test")

	err := db.RevokeCachePolicy(testImage, "no longer needed")
	if err != nil {
		t.Errorf("RevokeCachePolicy() error = %v", err)
	}

	// Verify policy is gone
	policy, _ := db.GetCachePolicy(testImage)
	if policy != nil {
		t.Error("expected policy to be revoked")
	}

	// Test revoking non-existent policy
	err = db.RevokeCachePolicy("nonexistent:latest", "test")
	if err == nil {
		t.Error("expected error when revoking non-existent policy")
	}
}

func TestGetPolicyAuditLog(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create some audit entries
	db.SetCachePolicy("nginx:latest", 3600, "test1")
	db.SetCachePolicy("redis:latest", 7200, "test2")
	db.RevokeCachePolicy("nginx:latest", "revoked")

	logs, err := db.GetPolicyAuditLog(10)
	if err != nil {
		t.Errorf("GetPolicyAuditLog() error = %v", err)
	}

	if len(logs) < 3 {
		t.Errorf("expected at least 3 audit entries, got %d", len(logs))
	}

	// Verify audit log contains expected actions
	actions := make(map[string]bool)
	for _, log := range logs {
		actions[log.Action] = true
	}

	if !actions["created"] {
		t.Error("expected 'created' action in audit log")
	}
	if !actions["revoked"] {
		t.Error("expected 'revoked' action in audit log")
	}
}

func TestGetAllActivePolicies(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Set up test data
	db.SetCachePolicy("active1:latest", 3600, "active")
	db.SetCachePolicy("active2:latest", 7200, "active")
	db.SetCachePolicy("expired:latest", -1, "expired")

	policies, err := db.GetAllActivePolicies()
	if err != nil {
		t.Errorf("GetAllActivePolicies() error = %v", err)
	}

	// Should only get active policies
	if len(policies) != 2 {
		t.Errorf("expected 2 active policies, got %d", len(policies))
	}

	// Verify no expired policies
	for _, p := range policies {
		if p.Image == "expired:latest" {
			t.Error("expired policy should not be in active policies")
		}
	}
}

func TestPolicyReplacement(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	image := "nginx:latest"

	// Set initial policy
	db.SetCachePolicy(image, 3600, "initial")
	policy1, _ := db.GetCachePolicy(image)

	// Replace with new policy
	db.SetCachePolicy(image, 7200, "updated")
	policy2, _ := db.GetCachePolicy(image)

	if policy1.TTL == policy2.TTL {
		t.Error("expected TTL to be updated")
	}

	if policy2.TTL != 7200 {
		t.Errorf("expected TTL 7200, got %d", policy2.TTL)
	}
}

func BenchmarkSetCachePolicy(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "registry-mirror-bench-*")
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)
	db, _ := NewDB()
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.SetCachePolicy("test:latest", 3600, "benchmark")
	}
}

func BenchmarkGetCachePolicy(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "registry-mirror-bench-*")
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)
	db, _ := NewDB()
	defer db.Close()

	db.SetCachePolicy("test:latest", 3600, "benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetCachePolicy("test:latest")
	}
}
