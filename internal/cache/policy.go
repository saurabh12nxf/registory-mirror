package cache

import (
	"fmt"

	"github.com/saurabh12nxf/registry-mirror/internal/storage"
)

type PolicyType string

const (
	PolicyLRU  PolicyType = "LRU"
	PolicyFIFO PolicyType = "FIFO"
)

type Manager struct {
	db      *storage.DB
	maxSize int64 // in bytes
	policy  PolicyType
}

func NewManager(db *storage.DB, maxSizeMB int64, policy PolicyType) *Manager {
	return &Manager{
		db:      db,
		maxSize: maxSizeMB * 1024 * 1024,
		policy:  policy,
	}
}

// Clean returns a list of images that should be deleted to free up space
// Note: In this version we just identify them, actual deletion would require registry API deletion support
func (m *Manager) Clean() ([]string, int64, error) {
	// 1. Get current usage
	stats, err := m.db.GetAggregatedStats()
	if err != nil {
		return nil, 0, err
	}

	if stats.TotalBytes <= m.maxSize {
		return nil, 0, nil // Under limit, no action needed
	}

	// 2. We are over limit, find what to delete
	bytesToFree := stats.TotalBytes - m.maxSize

	// Get all images sorted by last access (oldest first for LRU)
	records, err := m.db.GetRecentSyncs(1000) // simplified: just get recent ones
	if err != nil {
		return nil, 0, err
	}

	// Sort based on policy
	// Note: GetRecentSyncs returns Newest -> Oldest.
	// For LRU we want to delete Oldest (end of list), but "RecentSyncs" might not capture unique images correctly if there are duplicates.
	// For a real LRU implementation we'd need a "LastAccess" table.
	// We'll stick to a simple strategy: Delete the oldest sync records.

	// Actually, let's just implement a simple "suggest cleanup" for now since we can't easily delete from registry in this simple CLI.

	var candidates []string
	var predictedFreed int64 = 0

	// Walk backwards (Oldest first)
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]
		candidates = append(candidates, rec.Image)
		predictedFreed += rec.Bytes

		if predictedFreed >= bytesToFree {
			break
		}
	}

	return candidates, predictedFreed, nil
}

func (m *Manager) EnforcePolicy() error {
	candidates, freed, err := m.Clean()
	if err != nil {
		return err
	}

	if len(candidates) > 0 {
		fmt.Printf("🧹 Cache policy (%s) triggered. Suggested cleanup:\n", m.policy)
		for _, img := range candidates {
			fmt.Printf("   - %s\n", img)
		}
		fmt.Printf("   (Would free approx %.2f MB)\n", float64(freed)/(1024*1024))
	}
	return nil
}
