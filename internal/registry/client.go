package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	MediaType     string  `json:"mediaType"`
	Config        Layer   `json:"config"`
	Layers        []Layer `json:"layers"`
}

type Layer struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// NewClient creates a new registry client for the given registry URL
func NewClient(registryURL string) *Client {
	return &Client{
		baseURL:    registryURL,
		httpClient: &http.Client{},
	}
}

// getDockerHubToken obtains an authentication token from Docker Hub
func (c *Client) getDockerHubToken(ctx context.Context, image string) (string, error) {
	parts := strings.Split(image, ":")
	name := parts[0]

	// Add library/ prefix for official images without namespace
	if !strings.Contains(name, "/") {
		name = "library/" + name
	}

	tokenURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", name)

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.Token, nil
}

// GetManifest fetches the image manifest from the registry
func (c *Client) GetManifest(ctx context.Context, image string) (*Manifest, error) {
	parts := strings.Split(image, ":")
	name := parts[0]
	tag := "latest"
	if len(parts) > 1 {
		tag = parts[1]
	}

	// Add library/ prefix for official images
	if !strings.Contains(name, "/") {
		name = "library/" + name
	}

	// Get authentication token
	token, err := c.getDockerHubToken(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	url := fmt.Sprintf("https://registry-1.docker.io/v2/%s/manifests/%s", name, tag)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get manifest: %s (status: %d)", string(body), resp.StatusCode)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// PullLayer downloads a specific layer
func (c *Client) PullLayer(ctx context.Context, image, digest string) (io.ReadCloser, error) {
	parts := strings.Split(image, ":")
	name := parts[0]

	// Add library/ prefix for official images
	if !strings.Contains(name, "/") {
		name = "library/" + name
	}

	// Get authentication token
	token, err := c.getDockerHubToken(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	url := fmt.Sprintf("https://registry-1.docker.io/v2/%s/blobs/%s", name, digest)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to pull layer: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// PushLayer uploads a layer to the local registry
func (c *Client) PushLayer(ctx context.Context, image, digest string, data io.Reader) error {
	parts := strings.Split(image, ":")
	name := parts[0]

	url := fmt.Sprintf("http://%s/v2/%s/blobs/%s", c.baseURL, name, digest)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, data)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to push layer: status %d", resp.StatusCode)
	}

	return nil
}
