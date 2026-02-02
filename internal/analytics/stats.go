package analytics

import (
	"fmt"

	"github.com/saurabh12nxf/registry-mirror/internal/storage"
)

type Analyzer struct {
	db *storage.DB
}

func NewAnalyzer(db *storage.DB) *Analyzer {
	return &Analyzer{db: db}
}

type Report struct {
	TotalImages    int
	TotalBandwidth string
	TimeSaved      string
	AvgSpeed       string
}

func (a *Analyzer) GenerateReport() (*Report, error) {
	stats, err := a.db.GetAggregatedStats()
	if err != nil {
		return nil, err
	}

	// Calculate simpler metrics for display
	// Assume without mirror we get 5MB/s (typical home internet)
	// With mirror we get 100MB/s (local network)
	// Time saved = (Size / 5MBps) - ActualDuration

	totalMB := float64(stats.TotalBytes) / (1024 * 1024)
	estimatedTimeWithoutMirror := totalMB / 5.0 // seconds
	actualTime := stats.TotalDuration

	timeSaved := estimatedTimeWithoutMirror - actualTime
	if timeSaved < 0 {
		timeSaved = 0
	}

	avgSpeed := 0.0
	if stats.TotalDuration > 0 {
		avgSpeed = totalMB / stats.TotalDuration
	}

	return &Report{
		TotalImages:    stats.UniqueImages,
		TotalBandwidth: fmt.Sprintf("%.2f GB", totalMB/1024),
		TimeSaved:      fmt.Sprintf("%.1f min", timeSaved/60),
		AvgSpeed:       fmt.Sprintf("%.1f MB/s", avgSpeed),
	}, nil
}
