package storage

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNewDB(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("expected non-nil database")
	}

	if db.conn == nil {
		t.Fatal("expected non-nil connection")
	}
}

func TestRecordSync(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name     string
		image    string
		status   string
		bytes    int64
		duration float64
		wantErr  bool
	}{
		{
			name:     "successful sync",
			image:    "nginx:latest",
			status:   "completed",
			bytes:    1024000,
			duration: 5.5,
			wantErr:  false,
		},
		{
			name:     "failed sync",
			image:    "invalid:latest",
			status:   "failed",
			bytes:    0,
			duration: 0.1,
			wantErr:  false,
		},
		{
			name:     "large image",
			image:    "tensorflow:latest",
			status:   "completed",
			bytes:    1073741824, // 1GB
			duration: 120.5,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.RecordSync(tt.image, tt.status, tt.bytes, tt.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordSync() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify record was created
				record, err := db.GetLatestSync(tt.image)
				if err != nil {
					t.Errorf("GetLatestSync() error = %v", err)
				}
				if record == nil {
					t.Error("expected record to be created")
				}
				if record.Image != tt.image {
					t.Errorf("expected image %s, got %s", tt.image, record.Image)
				}
				if record.Status != tt.status {
					t.Errorf("expected status %s, got %s", tt.status, record.Status)
				}
			}
		})
	}
}

func TestGetRecentSyncs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test data
	testData := []struct {
		image  string
		status string
		bytes  int64
	}{
		{"nginx:latest", "completed", 1000},
		{"redis:latest", "completed", 2000},
		{"alpine:latest", "completed", 500},
		{"postgres:latest", "failed", 0},
		{"mysql:latest", "completed", 3000},
	}

	for _, td := range testData {
		db.RecordSync(td.image, td.status, td.bytes, 1.0)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	tests := []struct {
		name      string
		limit     int
		wantCount int
	}{
		{
			name:      "get 3 recent",
			limit:     3,
			wantCount: 3,
		},
		{
			name:      "get all",
			limit:     10,
			wantCount: 5,
		},
		{
			name:      "get 1",
			limit:     1,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := db.GetRecentSyncs(tt.limit)
			if err != nil {
				t.Errorf("GetRecentSyncs() error = %v", err)
			}

			if len(records) != tt.wantCount {
				t.Errorf("expected %d records, got %d", tt.wantCount, len(records))
			}

			// Verify records are in descending order by timestamp
			for i := 1; i < len(records); i++ {
				if records[i].Timestamp.After(records[i-1].Timestamp) {
					t.Error("records not in descending timestamp order")
				}
			}
		})
	}
}

func TestGetLatestSync(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testImage := "nginx:latest"

	// Insert multiple records for same image
	db.RecordSync(testImage, "completed", 1000, 1.0)
	time.Sleep(10 * time.Millisecond)
	db.RecordSync(testImage, "completed", 2000, 2.0)
	time.Sleep(10 * time.Millisecond)
	db.RecordSync(testImage, "failed", 0, 0.5)

	record, err := db.GetLatestSync(testImage)
	if err != nil {
		t.Errorf("GetLatestSync() error = %v", err)
	}

	if record == nil {
		t.Fatal("expected non-nil record")
	}

	// Should get the most recent (failed) record
	if record.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", record.Status)
	}

	// Test non-existent image
	record, err = db.GetLatestSync("nonexistent:latest")
	if err != nil {
		t.Errorf("GetLatestSync() error = %v", err)
	}
	if record != nil {
		t.Error("expected nil record for non-existent image")
	}
}

func TestGetAggregatedStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test data
	testData := []struct {
		image    string
		status   string
		bytes    int64
		duration float64
	}{
		{"nginx:latest", "completed", 1000, 1.0},
		{"nginx:latest", "completed", 1500, 1.5},
		{"redis:latest", "completed", 2000, 2.0},
		{"alpine:latest", "failed", 0, 0.0},
		{"postgres:latest", "completed", 3000, 3.0},
	}

	for _, td := range testData {
		db.RecordSync(td.image, td.status, td.bytes, td.duration)
	}

	stats, err := db.GetAggregatedStats()
	if err != nil {
		t.Errorf("GetAggregatedStats() error = %v", err)
	}

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	// Should only count completed syncs
	expectedCount := 4
	if stats.TotalCount != expectedCount {
		t.Errorf("expected count %d, got %d", expectedCount, stats.TotalCount)
	}

	expectedBytes := int64(1000 + 1500 + 2000 + 3000)
	if stats.TotalBytes != expectedBytes {
		t.Errorf("expected bytes %d, got %d", expectedBytes, stats.TotalBytes)
	}

	expectedDuration := 1.0 + 1.5 + 2.0 + 3.0
	if stats.TotalDuration != expectedDuration {
		t.Errorf("expected duration %.1f, got %.1f", expectedDuration, stats.TotalDuration)
	}

	// 3 unique images (nginx, redis, postgres - alpine failed)
	expectedUnique := 3
	if stats.UniqueImages != expectedUnique {
		t.Errorf("expected %d unique images, got %d", expectedUnique, stats.UniqueImages)
	}
}

func TestDatabaseConcurrency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// First, insert some test data
	for i := 0; i < 10; i++ {
		image := fmt.Sprintf("concurrent-test-%d:latest", i)
		err := db.RecordSync(image, "completed", int64(i*1000), float64(i))
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Test concurrent READS (SQLite handles concurrent reads well)
	numGoroutines := 10
	errors := make(chan error, numGoroutines)
	results := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			records, err := db.GetRecentSyncs(20)
			if err != nil {
				errors <- err
				return
			}
			results <- len(records)
			errors <- nil
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		if err := <-errors; err != nil {
			t.Errorf("GetRecentSyncs() error in goroutine: %v", err)
		}
		<-results
	}

	// Verify we can still read after concurrent operations
	records, err := db.GetRecentSyncs(100)
	if err != nil {
		t.Fatalf("GetRecentSyncs() error = %v", err)
	}

	if len(records) < 10 {
		t.Errorf("expected at least 10 records, got %d", len(records))
	}
}

func TestDatabaseClose(t *testing.T) {
	db, cleanup := setupTestDB(t)

	err := db.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Cleanup should still work
	cleanup()
}

func BenchmarkRecordSync(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "registry-mirror-bench-*")
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)
	db, _ := NewDB()
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.RecordSync("nginx:latest", "completed", 1024000, 5.5)
	}
}

func BenchmarkGetRecentSyncs(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "registry-mirror-bench-*")
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)
	db, _ := NewDB()
	defer db.Close()

	// Insert test data
	for i := 0; i < 100; i++ {
		db.RecordSync("nginx:latest", "completed", 1024000, 5.5)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetRecentSyncs(10)
	}
}

func BenchmarkGetAggregatedStats(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "registry-mirror-bench-*")
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)
	db, _ := NewDB()
	defer db.Close()

	// Insert test data
	for i := 0; i < 100; i++ {
		db.RecordSync("nginx:latest", "completed", 1024000, 5.5)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetAggregatedStats()
	}
}
