package mirror

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/saurabh12nxf/registry-mirror/internal/registry"
)

type Syncer struct {
	localRegistry string
	parallelism   int
	client        *registry.Client
}

type SyncProgress struct {
	Image        string
	TotalLayers  int
	SyncedLayers int
	BytesTotal   int64
	BytesSynced  int64
	StartTime    time.Time
}

func NewSyncer(localRegistry string, parallelism int) *Syncer {
	return &Syncer{
		localRegistry: localRegistry,
		parallelism:   parallelism,
		client:        registry.NewClient(localRegistry),
	}
}

func (s *Syncer) Sync(image string, force bool) error {
	ctx := context.Background()

	// Get manifest from Docker Hub
	manifest, err := s.client.GetManifest(ctx, image)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	fmt.Printf("📦 Found %d layers to sync\n", len(manifest.Layers))

	progress := &SyncProgress{
		Image:       image,
		TotalLayers: len(manifest.Layers),
		StartTime:   time.Now(),
	}

	for _, layer := range manifest.Layers {
		progress.BytesTotal += layer.Size
	}

	// Sync layers in parallel
	if err := s.syncLayers(ctx, image, manifest.Layers, progress); err != nil {
		return err
	}

	elapsed := time.Since(progress.StartTime)
	fmt.Printf("⏱️  Completed in %s (%.2f MB synced)\n",
		elapsed.Round(time.Second),
		float64(progress.BytesSynced)/(1024*1024))

	return nil
}

func (s *Syncer) syncLayers(ctx context.Context, image string, layers []registry.Layer, progress *SyncProgress) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(layers))
	semaphore := make(chan struct{}, s.parallelism)

	for i, layer := range layers {
		wg.Add(1)
		go func(idx int, l registry.Layer) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := s.syncLayer(ctx, image, l, idx+1, progress.TotalLayers); err != nil {
				errChan <- err
			} else {
				progress.SyncedLayers++
				progress.BytesSynced += l.Size
			}
		}(i, layer)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}

func (s *Syncer) syncLayer(ctx context.Context, image string, layer registry.Layer, current, total int) error {
	fmt.Printf("  [%d/%d] Syncing layer %s (%.2f MB)...\n",
		current, total, layer.Digest[:12], float64(layer.Size)/(1024*1024))

	// Pull from Docker Hub
	data, err := s.client.PullLayer(ctx, image, layer.Digest)
	if err != nil {
		return fmt.Errorf("failed to pull layer: %w", err)
	}
	defer data.Close()

	// Push to local registry
	if err := s.client.PushLayer(ctx, image, layer.Digest, data); err != nil {
		return fmt.Errorf("failed to push layer: %w", err)
	}

	return nil
}

// Helper to copy data and track progress
func copyWithProgress(dst io.Writer, src io.Reader, size int64) (int64, error) {
	// Simple copy for now, can add progress bar later
	return io.Copy(dst, src)
}
