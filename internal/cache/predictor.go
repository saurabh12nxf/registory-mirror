package cache

import (
	"github.com/saurabh12nxf/registry-mirror/internal/storage"
)

type Predictor struct {
	db *storage.DB
}

func NewPredictor(db *storage.DB) *Predictor {
	return &Predictor{db: db}
}

type PopularImage struct {
	Name      string
	PullCount int
}

// PredictTopImages analyzes usage patterns to suggest what should be mirrored
func (p *Predictor) PredictTopImages(limit int) ([]PopularImage, error) {
	// In a real scenario, this would check a 'pulls' table to see what is requested often but NOT yet mirrored.
	// Since we only track what we've already synced, we'll simulate "Discovery" by checking for
	// recurrent patterns or just returning a static list of "Must Have" images for this version.

	// Let's implement a feature that suggests images based on hardcoded "Popular" lists for now,
	// intersecting with what we already have.

	commonImages := []string{
		"nginx:latest",
		"alpine:latest",
		"ubuntu:latest",
		"postgres:latest",
		"redis:latest",
		"node:lts",
		"python:3.9",
		"golang:latest",
		"mysql:8.0",
		"mongo:latest",
	}

	// Filter out what we already have synced recently
	// This makes it "smart" - it won't suggest what you already have
	existing, err := p.db.GetRecentSyncs(100)
	if err != nil {
		return nil, err
	}

	have := make(map[string]bool)
	for _, rec := range existing {
		if rec.Status == "completed" {
			have[rec.Image] = true
		}
	}

	var suggestions []PopularImage
	for _, img := range commonImages {
		if !have[img] {
			suggestions = append(suggestions, PopularImage{Name: img, PullCount: 0})
		}
	}

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}
