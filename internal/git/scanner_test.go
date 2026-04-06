package git

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "git-scanner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestScanDockerfile(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create test Dockerfile
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	dockerfileContent := `FROM nginx:latest
FROM node:18-alpine AS builder
FROM postgres:15
# Comment line
FROM redis:7-alpine
`
	err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)
	if err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	scanner := NewScanner(tmpDir)
	images, err := scanner.scanDockerfile(dockerfilePath)
	if err != nil {
		t.Fatalf("scanDockerfile() error = %v", err)
	}

	// Should find 3 images (node:18-alpine is skipped as it's a build stage with AS)
	expectedImages := []string{"nginx:latest", "postgres:15", "redis:7-alpine"}

	if len(images) != len(expectedImages) {
		t.Errorf("expected %d images, got %d", len(expectedImages), len(images))
	}

	// Verify image names
	for i, expected := range expectedImages {
		if i < len(images) && images[i].Image != expected {
			t.Errorf("expected image %s, got %s", expected, images[i].Image)
		}
	}
}

func TestScanKubernetesManifest(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create test Kubernetes manifest
	manifestPath := filepath.Join(tmpDir, "deployment.yaml")
	manifestContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.21
      - name: sidecar
        image: busybox:latest
`
	err := os.WriteFile(manifestPath, []byte(manifestContent), 0644)
	if err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	scanner := NewScanner(tmpDir)
	images, err := scanner.scanKubernetesManifest(manifestPath)
	if err != nil {
		t.Fatalf("scanKubernetesManifest() error = %v", err)
	}

	if len(images) != 2 {
		t.Errorf("expected 2 images, got %d", len(images))
	}

	expectedImages := map[string]bool{
		"nginx:1.21":     true,
		"busybox:latest": true,
	}

	for _, img := range images {
		if !expectedImages[img.Image] {
			t.Errorf("unexpected image: %s", img.Image)
		}
	}
}

func TestScanDockerCompose(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create test docker-compose file
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	composeContent := `version: '3.8'
services:
  web:
    image: nginx:alpine
  db:
    image: postgres:14
  cache:
    image: redis:7
`
	err := os.WriteFile(composePath, []byte(composeContent), 0644)
	if err != nil {
		t.Fatalf("failed to write docker-compose: %v", err)
	}

	scanner := NewScanner(tmpDir)
	images, err := scanner.scanDockerCompose(composePath)
	if err != nil {
		t.Fatalf("scanDockerCompose() error = %v", err)
	}

	if len(images) != 3 {
		t.Errorf("expected 3 images, got %d", len(images))
	}
}

func TestScanRepository(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create multiple files
	dockerfile := filepath.Join(tmpDir, "Dockerfile")
	os.WriteFile(dockerfile, []byte("FROM nginx:latest\n"), 0644)

	k8sDir := filepath.Join(tmpDir, "k8s")
	os.MkdirAll(k8sDir, 0755)
	manifest := filepath.Join(k8sDir, "deployment.yaml")
	os.WriteFile(manifest, []byte("image: redis:7\n"), 0644)

	compose := filepath.Join(tmpDir, "docker-compose.yml")
	os.WriteFile(compose, []byte("image: postgres:15\n"), 0644)

	scanner := NewScanner(tmpDir)
	images, err := scanner.ScanRepository()
	if err != nil {
		t.Fatalf("ScanRepository() error = %v", err)
	}

	if len(images) < 3 {
		t.Errorf("expected at least 3 images, got %d", len(images))
	}
}

func TestDeduplicateImages(t *testing.T) {
	scanner := NewScanner(".")

	images := []ImageReference{
		{Image: "nginx:latest", Source: "Dockerfile"},
		{Image: "nginx:latest", Source: "Kubernetes"},
		{Image: "redis:7", Source: "Dockerfile"},
		{Image: "nginx:latest", Source: "Docker Compose"},
	}

	unique := scanner.deduplicateImages(images)

	if len(unique) != 2 {
		t.Errorf("expected 2 unique images, got %d", len(unique))
	}

	imageMap := make(map[string]bool)
	for _, img := range unique {
		imageMap[img.Image] = true
	}

	if !imageMap["nginx:latest"] || !imageMap["redis:7"] {
		t.Error("deduplication failed")
	}
}

func TestGenerateBillOfMaterials(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create test files
	dockerfile := filepath.Join(tmpDir, "Dockerfile")
	os.WriteFile(dockerfile, []byte("FROM nginx:latest\n"), 0644)

	scanner := NewScanner(tmpDir)
	bom, err := scanner.GenerateBillOfMaterials()
	if err != nil {
		t.Fatalf("GenerateBillOfMaterials() error = %v", err)
	}

	if bom == "" {
		t.Error("expected non-empty BOM")
	}

	// Check BOM contains expected content
	if !contains(bom, "Container Image Bill of Materials") {
		t.Error("BOM missing title")
	}

	if !contains(bom, "nginx:latest") {
		t.Error("BOM missing image reference")
	}
}

func TestScanRepositorySkipsGitDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create .git directory with a Dockerfile (should be skipped)
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	gitDockerfile := filepath.Join(gitDir, "Dockerfile")
	os.WriteFile(gitDockerfile, []byte("FROM should-be-skipped:latest\n"), 0644)

	// Create normal Dockerfile
	dockerfile := filepath.Join(tmpDir, "Dockerfile")
	os.WriteFile(dockerfile, []byte("FROM nginx:latest\n"), 0644)

	scanner := NewScanner(tmpDir)
	images, err := scanner.ScanRepository()
	if err != nil {
		t.Fatalf("ScanRepository() error = %v", err)
	}

	// Should only find nginx, not the one in .git
	for _, img := range images {
		if img.Image == "should-be-skipped:latest" {
			t.Error(".git directory was not skipped")
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) &&
			(s[:len(substr)] == substr || contains(s[1:], substr)))
}

func BenchmarkScanDockerfile(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "bench-*")
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	content := "FROM nginx:latest\nFROM node:18\nFROM postgres:15\n"
	os.WriteFile(dockerfilePath, []byte(content), 0644)

	scanner := NewScanner(tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.scanDockerfile(dockerfilePath)
	}
}

func BenchmarkScanRepository(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "bench-*")
	defer os.RemoveAll(tmpDir)

	// Create multiple files
	for i := 0; i < 10; i++ {
		path := filepath.Join(tmpDir, "Dockerfile"+string(rune(i)))
		os.WriteFile(path, []byte("FROM nginx:latest\n"), 0644)
	}

	scanner := NewScanner(tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.ScanRepository()
	}
}
