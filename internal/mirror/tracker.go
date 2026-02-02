package mirror

import (
	"fmt"
	"time"

	"github.com/saurabh12nxf/registry-mirror/internal/storage"
)

type Tracker struct {
	db *storage.DB
}

func NewTracker(db *storage.DB) *Tracker {
	return &Tracker{db: db}
}

func (t *Tracker) TrackSyncStart(image string) error {
	// We record a 'pending' or 'started' state
	// For simplicity, we'll just log it to stdout or maybe insert a record if we want deeper tracking
	// In this simple version, we mainly care about the result
	return nil
}

func (t *Tracker) TrackSyncComplete(image string, bytes int64, duration time.Duration) error {
	return t.db.RecordSync(image, "completed", bytes, duration.Seconds())
}

func (t *Tracker) TrackSyncError(image string, err error) error {
	// Record 0 bytes and 0 duration for errors
	return t.db.RecordSync(image, fmt.Sprintf("failed: %v", err), 0, 0)
}

func (t *Tracker) GetLastStatus(image string) (string, time.Time, error) {
	rec, err := t.db.GetLatestSync(image)
	if err != nil {
		return "", time.Time{}, err
	}
	if rec == nil {
		return "never_synced", time.Time{}, nil
	}
	return rec.Status, rec.Timestamp, nil
}
