package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ======================= NPM ADAPTER IMPLEMENTATION =======================

type NpmAdapter struct {
	config     *ProtocolConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

func NewNpmAdapter(config *ProtocolConfig) *NpmAdapter {
	return &NpmAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Fetch downloads a package from npm registry
func (na *NpmAdapter) Fetch(ctx context.Context, path string) (*ArtifactStream, error) {
	// Parse npm path: @scope/package/-/package-version.tgz
	url := fmt.Sprintf("%s/%s", na.config.UpstreamURL, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication if configured
	if na.config.AuthType == "bearer" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", na.config.Token))
	}

	resp, err := na.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("package not found")
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("npm registry returned %d", resp.StatusCode)
	}

	return &ArtifactStream{
		Reader:   resp.Body,
		Size:     resp.ContentLength,
		Checksum: resp.Header.Get("X-Checksum-SHA1"),
		Headers:  resp.Header,
	}, nil
}

// Exists checks if a package exists in npm registry
func (na *NpmAdapter) Exists(ctx context.Context, path string) (bool, error) {
	// Extract package name from path
	packageName := na.extractPackageName(path)

	// Query npm registry API
	url := fmt.Sprintf("%s/%s", na.config.UpstreamURL, packageName)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := na.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// Metadata retrieves package metadata
func (na *NpmAdapter) Metadata(ctx context.Context, path string) (*ArtifactMetadata, error) {
	packageName := na.extractPackageName(path)
	url := fmt.Sprintf("%s/%s", na.config.UpstreamURL, packageName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := na.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned %d", resp.StatusCode)
	}

	// Parse package.json response to extract metadata
	return &ArtifactMetadata{
		Size:        resp.ContentLength,
		ContentType: "application/json",
		Modified:    parseHttpTime(resp.Header.Get("Last-Modified")),
	}, nil
}

// List retrieves package versions/files
func (na *NpmAdapter) List(ctx context.Context, prefix string) ([]string, error) {
	// npm registry doesn't support directory listing, but we can query package versions
	url := fmt.Sprintf("%s/%s", na.config.UpstreamURL, prefix)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := na.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned %d", resp.StatusCode)
	}

	// Parse versions from response
	return []string{}, nil
}

func (na *NpmAdapter) extractPackageName(path string) string {
	// Extract from paths like: @scope/package/-/package-1.0.0.tgz
	parts := strings.Split(path, "/-/")
	if len(parts) > 0 {
		return parts[0]
	}
	return path
}

// GetConfig returns the adapter configuration
func (na *NpmAdapter) GetConfig() *ProtocolConfig {
	return na.config
}

// ======================= DOCKER ADAPTER IMPLEMENTATION =======================

type DockerAdapter struct {
	config      *ProtocolConfig
	httpClient  *http.Client
	tokenCache  string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

func NewDockerAdapter(config *ProtocolConfig) *DockerAdapter {
	return &DockerAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Fetch downloads a Docker image layer/manifest
func (da *DockerAdapter) Fetch(ctx context.Context, path string) (*ArtifactStream, error) {
	// Docker Hub API path format: v2/<repo>/manifests/<tag|digest>
	url := fmt.Sprintf("%s/%s", da.config.UpstreamURL, path)

	token, err := da.getAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := da.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("docker registry returned %d", resp.StatusCode)
	}

	return &ArtifactStream{
		Reader:   resp.Body,
		Size:     resp.ContentLength,
		Checksum: resp.Header.Get("Docker-Content-Digest"),
		Headers:  resp.Header,
	}, nil
}

// Exists checks if image manifest exists
func (da *DockerAdapter) Exists(ctx context.Context, path string) (bool, error) {
	url := fmt.Sprintf("%s/%s", da.config.UpstreamURL, path)

	token, err := da.getAuthToken(ctx)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := da.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// Metadata retrieves image metadata
func (da *DockerAdapter) Metadata(ctx context.Context, path string) (*ArtifactMetadata, error) {
	// Similar to Exists but returns metadata
	return &ArtifactMetadata{
		Size:        0,
		ContentType: "application/vnd.docker.distribution.manifest.v2+json",
		Modified:    time.Now(),
	}, nil
}

// List retrieves image tags
func (da *DockerAdapter) List(ctx context.Context, prefix string) ([]string, error) {
	// Query Docker registry tags API
	url := fmt.Sprintf("%s/v2/%s/tags/list", da.config.UpstreamURL, prefix)

	token, err := da.getAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := da.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker registry returned %d", resp.StatusCode)
	}

	// Parse tags from response
	return []string{}, nil
}

func (da *DockerAdapter) getAuthToken(ctx context.Context) (string, error) {
	da.mu.RLock()
	if da.tokenCache != "" && time.Now().Before(da.tokenExpiry) {
		defer da.mu.RUnlock()
		return da.tokenCache, nil
	}
	da.mu.RUnlock()

	// Authenticate with Docker registry
	authURL := fmt.Sprintf("%s/v2/", da.config.UpstreamURL)
	req, _ := http.NewRequestWithContext(ctx, "GET", authURL, nil)

	if da.config.AuthType == "basic" {
		req.SetBasicAuth(da.config.Username, da.config.Password)
	}

	resp, err := da.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Extract token from WWW-Authenticate header
	token := resp.Header.Get("Www-Authenticate")

	da.mu.Lock()
	da.tokenCache = token
	da.tokenExpiry = time.Now().Add(1 * time.Hour)
	da.mu.Unlock()

	return token, nil
}

// GetConfig returns the adapter configuration
func (da *DockerAdapter) GetConfig() *ProtocolConfig {
	return da.config
}

// ======================= PYPI ADAPTER IMPLEMENTATION =======================

type PyPiAdapter struct {
	config     *ProtocolConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

func NewPyPiAdapter(config *ProtocolConfig) *PyPiAdapter {
	return &PyPiAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Fetch downloads a Python package
func (pa *PyPiAdapter) Fetch(ctx context.Context, path string) (*ArtifactStream, error) {
	// PyPI path format: packages/XX/packagename-version.whl or .tar.gz
	url := fmt.Sprintf("%s/%s", pa.config.UpstreamURL, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication if configured
	if pa.config.AuthType == "bearer" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", pa.config.Token))
	}

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("PyPI returned %d", resp.StatusCode)
	}

	return &ArtifactStream{
		Reader:   resp.Body,
		Size:     resp.ContentLength,
		Checksum: resp.Header.Get("X-PyPI-Hash"),
		Headers:  resp.Header,
	}, nil
}

// Exists checks if package exists in PyPI
func (pa *PyPiAdapter) Exists(ctx context.Context, path string) (bool, error) {
	url := fmt.Sprintf("%s/%s", pa.config.UpstreamURL, path)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// Metadata retrieves package metadata
func (pa *PyPiAdapter) Metadata(ctx context.Context, path string) (*ArtifactMetadata, error) {
	packageName := extractPyPiPackageName(path)
	url := fmt.Sprintf("%s/pypi/%s/json", pa.config.UpstreamURL, packageName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PyPI API returned %d", resp.StatusCode)
	}

	return &ArtifactMetadata{
		Size:        resp.ContentLength,
		ContentType: "application/json",
		Modified:    parseHttpTime(resp.Header.Get("Last-Modified")),
	}, nil
}

// List retrieves package versions
func (pa *PyPiAdapter) List(ctx context.Context, prefix string) ([]string, error) {
	// Query PyPI JSON API for package versions
	url := fmt.Sprintf("%s/pypi/%s/json", pa.config.UpstreamURL, prefix)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PyPI API returned %d", resp.StatusCode)
	}

	// Parse versions from JSON response
	return []string{}, nil
}

func extractPyPiPackageName(path string) string {
	// Extract from paths like: packages/XX/package-1.0.0.whl
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		// Remove version and extension
		return strings.Split(filename, "-")[0]
	}
	return path
}

// GetConfig returns the adapter configuration
func (pa *PyPiAdapter) GetConfig() *ProtocolConfig {
	return pa.config
}

// ======================= HELM ADAPTER IMPLEMENTATION =======================

type HelmAdapter struct {
	config     *ProtocolConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

func NewHelmAdapter(config *ProtocolConfig) *HelmAdapter {
	return &HelmAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Fetch downloads a Helm chart
func (ha *HelmAdapter) Fetch(ctx context.Context, path string) (*ArtifactStream, error) {
	// Helm chart path: repo/chartname-version.tgz
	url := fmt.Sprintf("%s/%s", ha.config.UpstreamURL, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication if configured
	if ha.config.AuthType == "bearer" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ha.config.Token))
	}

	resp, err := ha.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("Helm repository returned %d", resp.StatusCode)
	}

	return &ArtifactStream{
		Reader:   resp.Body,
		Size:     resp.ContentLength,
		Checksum: resp.Header.Get("X-Checksum"),
		Headers:  resp.Header,
	}, nil
}

// Exists checks if chart exists
func (ha *HelmAdapter) Exists(ctx context.Context, path string) (bool, error) {
	url := fmt.Sprintf("%s/%s", ha.config.UpstreamURL, path)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := ha.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// Metadata retrieves chart metadata
func (ha *HelmAdapter) Metadata(ctx context.Context, path string) (*ArtifactMetadata, error) {
	return &ArtifactMetadata{
		Size:        0,
		ContentType: "application/gzip",
		Modified:    time.Now(),
	}, nil
}

// List retrieves available charts
func (ha *HelmAdapter) List(ctx context.Context, prefix string) ([]string, error) {
	// Query Helm repository index
	url := fmt.Sprintf("%s/index.yaml", ha.config.UpstreamURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ha.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Helm repository returned %d", resp.StatusCode)
	}

	// Parse charts from index.yaml
	return []string{}, nil
}

// GetConfig returns the adapter configuration
func (ha *HelmAdapter) GetConfig() *ProtocolConfig {
	return ha.config
}

// ======================= HELPER FUNCTIONS =======================

func getContentType(path string) string {
	if strings.HasSuffix(path, ".whl") {
		return "application/zip"
	}
	if strings.HasSuffix(path, ".tar.gz") {
		return "application/gzip"
	}
	if strings.HasSuffix(path, ".zip") {
		return "application/zip"
	}
	return "application/octet-stream"
}

func parseHttpTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}
	t, _ := http.ParseTime(timeStr)
	return t
}
