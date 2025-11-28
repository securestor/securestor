package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// StringArray represents a PostgreSQL text array
type StringArray []string

func (sa *StringArray) Scan(value interface{}) error {
	// Use pq.Array for proper PostgreSQL array scanning
	var arr []string
	if err := pq.Array(&arr).Scan(value); err != nil {
		return err
	}
	*sa = StringArray(arr)
	return nil
}

func (sa StringArray) Value() (driver.Value, error) {
	// Use pq.Array for proper PostgreSQL array handling
	return pq.Array([]string(sa)).Value()
}

// PyPIPackage represents a Python package in PyPI format
type PyPIPackage struct {
	ID                     int64           `json:"id" db:"id"`
	Name                   string          `json:"name" db:"name"`
	NormalizedName         string          `json:"normalized_name" db:"normalized_name"`
	Author                 *string         `json:"author,omitempty" db:"author"`
	AuthorEmail            *string         `json:"author_email,omitempty" db:"author_email"`
	Maintainer             *string         `json:"maintainer,omitempty" db:"maintainer"`
	MaintainerEmail        *string         `json:"maintainer_email,omitempty" db:"maintainer_email"`
	HomePage               *string         `json:"home_page,omitempty" db:"home_page"`
	DownloadURL            *string         `json:"download_url,omitempty" db:"download_url"`
	Summary                *string         `json:"summary,omitempty" db:"summary"`
	Description            *string         `json:"description,omitempty" db:"description"`
	DescriptionContentType *string         `json:"description_content_type,omitempty" db:"description_content_type"`
	Keywords               *string         `json:"keywords,omitempty" db:"keywords"`
	License                *string         `json:"license,omitempty" db:"license"`
	Classifiers            StringArray     `json:"classifiers" db:"classifiers"`
	Platforms              StringArray     `json:"platforms" db:"platforms"`
	ProjectURLs            json.RawMessage `json:"project_urls" db:"project_urls"`
	RequiresDist           StringArray     `json:"requires_dist" db:"requires_dist"`
	RequiresPython         *string         `json:"requires_python,omitempty" db:"requires_python"`
	ProvidesExtra          StringArray     `json:"provides_extra" db:"provides_extra"`
	RepositoryID           *int64          `json:"repository_id,omitempty" db:"repository_id"`
	TenantID               *int64          `json:"tenant_id,omitempty" db:"tenant_id"`
	CreatedBy              *int64          `json:"created_by,omitempty" db:"created_by"`
	IsActive               bool            `json:"is_active" db:"is_active"`
	Yanked                 bool            `json:"yanked" db:"yanked"`
	YankedReason           *string         `json:"yanked_reason,omitempty" db:"yanked_reason"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at" db:"updated_at"`

	// Related data
	Versions   []PyPIPackageVersion `json:"versions,omitempty"`
	Repository *Repository          `json:"repository,omitempty"`
	Tenant     *Tenant              `json:"tenant,omitempty"`
}

// PyPIPackageVersion represents a specific version of a Python package
type PyPIPackageVersion struct {
	ID                     int64           `json:"id" db:"id"`
	PackageID              int64           `json:"package_id" db:"package_id"`
	Version                string          `json:"version" db:"version"`
	NormalizedVersion      string          `json:"normalized_version" db:"normalized_version"`
	Summary                *string         `json:"summary,omitempty" db:"summary"`
	Description            *string         `json:"description,omitempty" db:"description"`
	DescriptionContentType *string         `json:"description_content_type,omitempty" db:"description_content_type"`
	Author                 *string         `json:"author,omitempty" db:"author"`
	AuthorEmail            *string         `json:"author_email,omitempty" db:"author_email"`
	Maintainer             *string         `json:"maintainer,omitempty" db:"maintainer"`
	MaintainerEmail        *string         `json:"maintainer_email,omitempty" db:"maintainer_email"`
	HomePage               *string         `json:"home_page,omitempty" db:"home_page"`
	DownloadURL            *string         `json:"download_url,omitempty" db:"download_url"`
	Keywords               *string         `json:"keywords,omitempty" db:"keywords"`
	License                *string         `json:"license,omitempty" db:"license"`
	Classifiers            StringArray     `json:"classifiers" db:"classifiers"`
	Platforms              StringArray     `json:"platforms" db:"platforms"`
	ProjectURLs            json.RawMessage `json:"project_urls" db:"project_urls"`
	RequiresDist           StringArray     `json:"requires_dist" db:"requires_dist"`
	RequiresPython         *string         `json:"requires_python,omitempty" db:"requires_python"`
	ProvidesExtra          StringArray     `json:"provides_extra" db:"provides_extra"`
	UploadTime             time.Time       `json:"upload_time" db:"upload_time"`
	UploadedBy             *int64          `json:"uploaded_by,omitempty" db:"uploaded_by"`
	IsPrerelease           bool            `json:"is_prerelease" db:"is_prerelease"`
	IsYanked               bool            `json:"is_yanked" db:"is_yanked"`
	YankedReason           *string         `json:"yanked_reason,omitempty" db:"yanked_reason"`
	Metadata               json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at" db:"updated_at"`

	// Related data
	Package          *PyPIPackage               `json:"package,omitempty"`
	Files            []PyPIPackageFile          `json:"files,omitempty"`
	Dependencies     []PyPIPackageDependency    `json:"dependencies,omitempty"`
	SecurityScans    []PyPISecurityScan         `json:"security_scans,omitempty"`
	Vulnerabilities  []PyPIPackageVulnerability `json:"vulnerabilities,omitempty"`
	ComplianceChecks []PyPIComplianceCheck      `json:"compliance_checks,omitempty"`
}

// PyPIPackageFile represents a file associated with a package version (.whl, .tar.gz, etc.)
type PyPIPackageFile struct {
	ID               int64           `json:"id" db:"id"`
	PackageVersionID int64           `json:"package_version_id" db:"package_version_id"`
	Filename         string          `json:"filename" db:"filename"`
	OriginalFilename *string         `json:"original_filename,omitempty" db:"original_filename"`
	FileType         string          `json:"file_type" db:"file_type"` // 'wheel', 'sdist'
	PythonVersion    *string         `json:"python_version,omitempty" db:"python_version"`
	ABITag           *string         `json:"abi_tag,omitempty" db:"abi_tag"`
	PlatformTag      *string         `json:"platform_tag,omitempty" db:"platform_tag"`
	FileSize         int64           `json:"file_size" db:"file_size"`
	MD5Digest        *string         `json:"md5_digest,omitempty" db:"md5_digest"`
	SHA1Digest       *string         `json:"sha1_digest,omitempty" db:"sha1_digest"`
	SHA256Digest     string          `json:"sha256_digest" db:"sha256_digest"`
	Blake2256Digest  *string         `json:"blake2_256_digest,omitempty" db:"blake2_256_digest"`
	StoragePath      string          `json:"storage_path" db:"storage_path"`
	ContentType      *string         `json:"content_type,omitempty" db:"content_type"`
	Encoding         *string         `json:"encoding,omitempty" db:"encoding"`
	UploadTime       time.Time       `json:"upload_time" db:"upload_time"`
	UploadedBy       *int64          `json:"uploaded_by,omitempty" db:"uploaded_by"`
	DownloadCount    int64           `json:"download_count" db:"download_count"`
	LastDownloadedAt *time.Time      `json:"last_downloaded_at,omitempty" db:"last_downloaded_at"`
	IsAvailable      bool            `json:"is_available" db:"is_available"`
	Metadata         json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`

	// Related data
	PackageVersion *PyPIPackageVersion `json:"package_version,omitempty"`
}

// PyPIPackageDependency represents a dependency relationship
type PyPIPackageDependency struct {
	ID                int64     `json:"id" db:"id"`
	PackageVersionID  int64     `json:"package_version_id" db:"package_version_id"`
	DependencyType    string    `json:"dependency_type" db:"dependency_type"` // 'requires', 'provides', 'obsoletes'
	DependencyName    string    `json:"dependency_name" db:"dependency_name"`
	VersionSpec       *string   `json:"version_spec,omitempty" db:"version_spec"`
	Extra             *string   `json:"extra,omitempty" db:"extra"`
	EnvironmentMarker *string   `json:"environment_marker,omitempty" db:"environment_marker"`
	IsOptional        bool      `json:"is_optional" db:"is_optional"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// PyPISecurityScan represents a security scan performed on a package version
type PyPISecurityScan struct {
	ID                  int64           `json:"id" db:"id"`
	PackageVersionID    int64           `json:"package_version_id" db:"package_version_id"`
	ScanType            string          `json:"scan_type" db:"scan_type"` // 'safety', 'bandit', 'dependency_check'
	ScannerVersion      *string         `json:"scanner_version,omitempty" db:"scanner_version"`
	ScanStatus          string          `json:"scan_status" db:"scan_status"` // 'pending', 'running', 'completed', 'failed'
	VulnerabilityCount  int             `json:"vulnerability_count" db:"vulnerability_count"`
	HighSeverityCount   int             `json:"high_severity_count" db:"high_severity_count"`
	MediumSeverityCount int             `json:"medium_severity_count" db:"medium_severity_count"`
	LowSeverityCount    int             `json:"low_severity_count" db:"low_severity_count"`
	ScanResults         json.RawMessage `json:"scan_results" db:"scan_results"`
	ErrorMessage        *string         `json:"error_message,omitempty" db:"error_message"`
	StartedAt           time.Time       `json:"started_at" db:"started_at"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`

	// Related data
	PackageVersion  *PyPIPackageVersion        `json:"package_version,omitempty"`
	Vulnerabilities []PyPIPackageVulnerability `json:"vulnerabilities,omitempty"`
}

// PyPIPackageVulnerability represents a security vulnerability found in a package
type PyPIPackageVulnerability struct {
	ID               int64           `json:"id" db:"id"`
	PackageVersionID int64           `json:"package_version_id" db:"package_version_id"`
	SecurityScanID   int64           `json:"security_scan_id" db:"security_scan_id"`
	VulnerabilityID  *string         `json:"vulnerability_id,omitempty" db:"vulnerability_id"` // CVE, GHSA, etc.
	Title            *string         `json:"title,omitempty" db:"title"`
	Description      *string         `json:"description,omitempty" db:"description"`
	Severity         *string         `json:"severity,omitempty" db:"severity"` // 'critical', 'high', 'medium', 'low'
	CVSSScore        *float64        `json:"cvss_score,omitempty" db:"cvss_score"`
	CVSSVector       *string         `json:"cvss_vector,omitempty" db:"cvss_vector"`
	CWEIDs           StringArray     `json:"cwe_ids" db:"cwe_ids"`
	AffectedVersions StringArray     `json:"affected_versions" db:"affected_versions"`
	PatchedVersions  StringArray     `json:"patched_versions" db:"patched_versions"`
	PublishedAt      *time.Time      `json:"published_at,omitempty" db:"published_at"`
	UpdatedAtSource  *time.Time      `json:"updated_at_source,omitempty" db:"updated_at_source"`
	Source           *string         `json:"source,omitempty" db:"source"` // 'safety', 'osv', 'github'
	SourceURL        *string         `json:"source_url,omitempty" db:"source_url"`
	References       StringArray     `json:"references" db:"references"`
	Metadata         json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`

	// Related data
	PackageVersion *PyPIPackageVersion `json:"package_version,omitempty"`
	SecurityScan   *PyPISecurityScan   `json:"security_scan,omitempty"`
}

// PyPIComplianceCheck represents a compliance check result for a package
type PyPIComplianceCheck struct {
	ID               int64           `json:"id" db:"id"`
	PackageVersionID int64           `json:"package_version_id" db:"package_version_id"`
	PolicyName       string          `json:"policy_name" db:"policy_name"`
	CheckType        string          `json:"check_type" db:"check_type"` // 'license', 'security', 'quality', 'governance'
	Status           string          `json:"status" db:"status"`         // 'passed', 'failed', 'warning', 'pending'
	Result           json.RawMessage `json:"result" db:"result"`
	ErrorMessage     *string         `json:"error_message,omitempty" db:"error_message"`
	CheckedAt        time.Time       `json:"checked_at" db:"checked_at"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`

	// Related data
	PackageVersion *PyPIPackageVersion `json:"package_version,omitempty"`
}

// PyPIUploadRequest represents a request to upload a new package/version
type PyPIUploadRequest struct {
	// Metadata fields from setup.py/pyproject.toml
	Name                   string            `json:"name" form:"name"`
	Version                string            `json:"version" form:"version"`
	Summary                string            `json:"summary" form:"summary"`
	Description            string            `json:"description" form:"description"`
	DescriptionContentType string            `json:"description_content_type" form:"description_content_type"`
	Author                 string            `json:"author" form:"author"`
	AuthorEmail            string            `json:"author_email" form:"author_email"`
	Maintainer             string            `json:"maintainer" form:"maintainer"`
	MaintainerEmail        string            `json:"maintainer_email" form:"maintainer_email"`
	HomePage               string            `json:"home_page" form:"home_page"`
	DownloadURL            string            `json:"download_url" form:"download_url"`
	Keywords               string            `json:"keywords" form:"keywords"`
	License                string            `json:"license" form:"license"`
	Classifiers            []string          `json:"classifiers" form:"classifiers"`
	Platforms              []string          `json:"platforms" form:"platforms"`
	RequiresDist           []string          `json:"requires_dist" form:"requires_dist"`
	RequiresPython         string            `json:"requires_python" form:"requires_python"`
	ProvidesExtra          []string          `json:"provides_extra" form:"provides_extra"`
	ProjectURLs            map[string]string `json:"project_urls" form:"project_urls"`

	// File information
	Filename      string `json:"filename" form:"filename"`
	FileType      string `json:"filetype" form:"filetype"`
	PythonVersion string `json:"pyversion" form:"pyversion"`
	MD5Digest     string `json:"md5_digest" form:"md5_digest"`
	SHA256Digest  string `json:"sha256_digest" form:"sha256_digest"`

	// Upload metadata
	RepositoryID *int64 `json:"repository_id"`
	Comment      string `json:"comment" form:"comment"`
}

// PyPIPackageInfo represents package information in PEP 503 format
type PyPIPackageInfo struct {
	Name            string            `json:"name"`
	NormalizedName  string            `json:"normalized_name"`
	Summary         *string           `json:"summary,omitempty"`
	Description     *string           `json:"description,omitempty"`
	Keywords        *string           `json:"keywords,omitempty"`
	License         *string           `json:"license,omitempty"`
	HomePage        *string           `json:"home_page,omitempty"`
	Author          *string           `json:"author,omitempty"`
	AuthorEmail     *string           `json:"author_email,omitempty"`
	Maintainer      *string           `json:"maintainer,omitempty"`
	MaintainerEmail *string           `json:"maintainer_email,omitempty"`
	RequiresPython  *string           `json:"requires_python,omitempty"`
	Versions        []PyPIVersionInfo `json:"versions"`
	Yanked          bool              `json:"yanked"`
	YankedReason    *string           `json:"yanked_reason,omitempty"`
}

// PyPIVersionInfo represents version information in PEP 503 format
type PyPIVersionInfo struct {
	Version      string         `json:"version"`
	Files        []PyPIFileInfo `json:"files"`
	IsPrerelease bool           `json:"is_prerelease"`
	IsYanked     bool           `json:"is_yanked"`
	YankedReason *string        `json:"yanked_reason,omitempty"`
	UploadTime   time.Time      `json:"upload_time"`
}

// PyPIFileInfo represents file information in PEP 503 format
type PyPIFileInfo struct {
	Filename       string    `json:"filename"`
	URL            string    `json:"url"`
	Size           int64     `json:"size"`
	SHA256Digest   string    `json:"sha256_digest"`
	MD5Digest      *string   `json:"md5_digest,omitempty"`
	FileType       string    `json:"file_type"`
	PythonVersion  *string   `json:"python_version,omitempty"`
	RequiresPython *string   `json:"requires_python,omitempty"`
	UploadTime     time.Time `json:"upload_time"`
	HasSignature   bool      `json:"has_signature"`
}

// PyPIIndexResponse represents the simple API response for package index
type PyPIIndexResponse struct {
	Projects []PyPIProjectLink `json:"projects"`
	Meta     PyPIIndexMeta     `json:"meta"`
}

// PyPIProjectLink represents a project link in the simple index
type PyPIProjectLink struct {
	Name           string `json:"name"`
	NormalizedName string `json:"normalized_name"`
	URL            string `json:"url"`
}

// PyPIIndexMeta represents metadata for the simple index
type PyPIIndexMeta struct {
	APIVersion string `json:"api-version"`
	Repository string `json:"repository"`
}

// PyPISimpleResponse represents the simple API response for a specific package
type PyPISimpleResponse struct {
	Name  string         `json:"name"`
	Files []PyPIFileLink `json:"files"`
	Meta  PyPIIndexMeta  `json:"meta"`
}

// PyPIFileLink represents a file link in the simple API
type PyPIFileLink struct {
	Filename       string    `json:"filename"`
	URL            string    `json:"url"`
	SHA256         string    `json:"data-sha256,omitempty"`
	RequiresPython *string   `json:"data-requires-python,omitempty"`
	Yanked         bool      `json:"data-yanked,omitempty"`
	YankedReason   *string   `json:"data-yanked-reason,omitempty"`
	Size           int64     `json:"size,omitempty"`
	UploadTime     time.Time `json:"upload_time,omitempty"`
}

// PyPISearchResult represents search result for packages
type PyPISearchResult struct {
	Name        string  `json:"name"`
	Version     string  `json:"version"`
	Summary     *string `json:"summary"`
	Description *string `json:"description"`
	Keywords    *string `json:"keywords"`
	Author      *string `json:"author"`
	HomePage    *string `json:"home_page"`
	Score       float64 `json:"score"`
}

// PyPIStatistics represents package statistics
type PyPIStatistics struct {
	TotalPackages    int64                    `json:"total_packages"`
	TotalVersions    int64                    `json:"total_versions"`
	TotalFiles       int64                    `json:"total_files"`
	TotalDownloads   int64                    `json:"total_downloads"`
	RecentUploads    []PyPIRecentUpload       `json:"recent_uploads"`
	PopularPackages  []PyPIPopularPackage     `json:"popular_packages"`
	SecurityScans    PyPISecurityStatistics   `json:"security_scans"`
	ComplianceStatus PyPIComplianceStatistics `json:"compliance_status"`
}

// PyPIRecentUpload represents a recently uploaded package
type PyPIRecentUpload struct {
	PackageName string    `json:"package_name"`
	Version     string    `json:"version"`
	UploadTime  time.Time `json:"upload_time"`
	UploadedBy  *string   `json:"uploaded_by"`
}

// PyPIPopularPackage represents a popular package by download count
type PyPIPopularPackage struct {
	PackageName   string `json:"package_name"`
	DownloadCount int64  `json:"download_count"`
	VersionCount  int    `json:"version_count"`
}

// PyPISecurityStatistics represents security scan statistics
type PyPISecurityStatistics struct {
	TotalScans         int64 `json:"total_scans"`
	PendingScans       int64 `json:"pending_scans"`
	CompletedScans     int64 `json:"completed_scans"`
	FailedScans        int64 `json:"failed_scans"`
	VulnerablePackages int64 `json:"vulnerable_packages"`
	CriticalVulns      int64 `json:"critical_vulnerabilities"`
	HighVulns          int64 `json:"high_vulnerabilities"`
	MediumVulns        int64 `json:"medium_vulnerabilities"`
	LowVulns           int64 `json:"low_vulnerabilities"`
}

// PyPIComplianceStatistics represents compliance check statistics
type PyPIComplianceStatistics struct {
	TotalChecks    int64   `json:"total_checks"`
	PassedChecks   int64   `json:"passed_checks"`
	FailedChecks   int64   `json:"failed_checks"`
	WarningChecks  int64   `json:"warning_checks"`
	PendingChecks  int64   `json:"pending_checks"`
	ComplianceRate float64 `json:"compliance_rate"`
}
