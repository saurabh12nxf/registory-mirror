package git

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Scanner struct {
	repoPath string
}

type ImageReference struct {
	Image      string
	Source     string // Dockerfile, k8s manifest, docker-compose, etc.
	FilePath   string
	LineNumber int
}

func NewScanner(repoPath string) *Scanner {
	return &Scanner{
		repoPath: repoPath,
	}
}

// ScanRepository scans the entire repository for container image references
func (s *Scanner) ScanRepository() ([]ImageReference, error) {
	var images []ImageReference

	err := filepath.Walk(s.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip node_modules and other common directories
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == "vendor") {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		// Scan different file types
		switch {
		case strings.HasSuffix(path, "Dockerfile") || strings.Contains(path, "Dockerfile."):
			refs, err := s.scanDockerfile(path)
			if err == nil {
				images = append(images, refs...)
			}
		case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml"):
			refs, err := s.scanKubernetesManifest(path)
			if err == nil {
				images = append(images, refs...)
			}
		case strings.HasSuffix(path, "docker-compose.yml") || strings.HasSuffix(path, "docker-compose.yaml"):
			refs, err := s.scanDockerCompose(path)
			if err == nil {
				images = append(images, refs...)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.deduplicateImages(images), nil
}

// scanDockerfile extracts image references from Dockerfile
func (s *Scanner) scanDockerfile(path string) ([]ImageReference, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var images []ImageReference
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Regex for FROM instruction
	fromRegex := regexp.MustCompile(`(?i)^FROM\s+(?:--platform=[^\s]+\s+)?([^\s]+)`)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		matches := fromRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			image := matches[1]
			// Skip build stages (AS stage_name)
			if !strings.Contains(strings.ToUpper(line), " AS ") {
				images = append(images, ImageReference{
					Image:      image,
					Source:     "Dockerfile",
					FilePath:   path,
					LineNumber: lineNum,
				})
			}
		}
	}

	return images, scanner.Err()
}

// scanKubernetesManifest extracts image references from Kubernetes YAML files
func (s *Scanner) scanKubernetesManifest(path string) ([]ImageReference, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var images []ImageReference
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Regex for image: field in Kubernetes manifests
	imageRegex := regexp.MustCompile(`^\s*image:\s*["']?([^\s"']+)["']?`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		matches := imageRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			image := matches[1]
			images = append(images, ImageReference{
				Image:      image,
				Source:     "Kubernetes",
				FilePath:   path,
				LineNumber: lineNum,
			})
		}
	}

	return images, scanner.Err()
}

// scanDockerCompose extracts image references from docker-compose files
func (s *Scanner) scanDockerCompose(path string) ([]ImageReference, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var images []ImageReference
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Regex for image: field in docker-compose
	imageRegex := regexp.MustCompile(`^\s*image:\s*["']?([^\s"']+)["']?`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		matches := imageRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			image := matches[1]
			images = append(images, ImageReference{
				Image:      image,
				Source:     "Docker Compose",
				FilePath:   path,
				LineNumber: lineNum,
			})
		}
	}

	return images, scanner.Err()
}

// deduplicateImages removes duplicate image references
func (s *Scanner) deduplicateImages(images []ImageReference) []ImageReference {
	seen := make(map[string]bool)
	var unique []ImageReference

	for _, img := range images {
		if !seen[img.Image] {
			seen[img.Image] = true
			unique = append(unique, img)
		}
	}

	return unique
}

// GenerateBillOfMaterials creates a BOM for all container images
func (s *Scanner) GenerateBillOfMaterials() (string, error) {
	images, err := s.ScanRepository()
	if err != nil {
		return "", err
	}

	var bom strings.Builder
	bom.WriteString("# Container Image Bill of Materials\n\n")
	bom.WriteString(fmt.Sprintf("Repository: %s\n", s.repoPath))
	bom.WriteString(fmt.Sprintf("Total Images: %d\n\n", len(images)))
	bom.WriteString("## Images\n\n")

	// Group by source type
	bySource := make(map[string][]ImageReference)
	for _, img := range images {
		bySource[img.Source] = append(bySource[img.Source], img)
	}

	for source, refs := range bySource {
		bom.WriteString(fmt.Sprintf("### %s\n\n", source))
		for _, ref := range refs {
			relPath, _ := filepath.Rel(s.repoPath, ref.FilePath)
			bom.WriteString(fmt.Sprintf("- `%s`\n", ref.Image))
			bom.WriteString(fmt.Sprintf("  - File: %s (line %d)\n", relPath, ref.LineNumber))
		}
		bom.WriteString("\n")
	}

	return bom.String(), nil
}
