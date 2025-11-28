package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

type ScanRepository struct {
	db *sql.DB
}

func NewScanRepository(db *sql.DB) *ScanRepository {
	return &ScanRepository{db: db}
}

// Create creates a new scan record
func (r *ScanRepository) Create(scan *models.SecurityScan) error {
	query := `
		INSERT INTO security_scans (
			tenant_id, artifact_id, status, scan_type, priority, 
			vulnerability_scan, malware_scan, license_scan, dependency_scan,
			initiated_by, started_at, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING scan_id, created_at, updated_at`

	metadataJSON, _ := json.Marshal(scan.Metadata)

	// Get tenant_id from artifact if not provided
	var tenantID uuid.UUID
	if scan.TenantID == uuid.Nil {
		// Try to get from artifact
		err := r.db.QueryRow(`SELECT tenant_id FROM artifacts WHERE artifact_id = $1`, scan.ArtifactID).Scan(&tenantID)
		if err != nil {
			return fmt.Errorf("failed to get tenant_id: %w", err)
		}
	} else {
		tenantID = scan.TenantID
	}

	err := r.db.QueryRow(
		query,
		tenantID,
		scan.ArtifactID,
		scan.Status,
		scan.ScanType,
		scan.Priority,
		scan.VulnerabilityScan,
		scan.MalwareScan,
		scan.LicenseScan,
		scan.DependencyScan,
		scan.InitiatedBy,
		scan.StartedAt,
		metadataJSON,
		time.Now(),
		time.Now(),
	).Scan(&scan.ID, &scan.CreatedAt, &scan.UpdatedAt)

	return err
}

// Update updates an existing scan record
func (r *ScanRepository) Update(scan *models.SecurityScan) error {
	query := `
		UPDATE security_scans SET
			status = $2,
			completed_at = $3,
			duration = $4,
			error_message = $5,
			metadata = $6,
			updated_at = $7
		WHERE scan_id = $1`

	metadataJSON, _ := json.Marshal(scan.Metadata)

	_, err := r.db.Exec(
		query,
		scan.ID,
		scan.Status,
		scan.CompletedAt,
		scan.Duration,
		scan.ErrorMessage,
		metadataJSON,
		time.Now(),
	)

	return err
}

// GetByID retrieves a scan by its ID
func (r *ScanRepository) GetByID(scanID int64) (*models.SecurityScan, error) {
	query := `
		SELECT scan_id, artifact_id, status, scan_type, priority,
			   vulnerability_scan, malware_scan, license_scan, dependency_scan,
			   initiated_by, started_at, completed_at, duration, error_message,
			   metadata, created_at, updated_at
		FROM security_scans
		WHERE scan_id = $1`

	scan := &models.SecurityScan{}
	var metadataJSON []byte
	var completedAt sql.NullTime
	var duration sql.NullInt64
	var errorMessage sql.NullString

	err := r.db.QueryRow(query, scanID).Scan(
		&scan.ID,
		&scan.ArtifactID,
		&scan.Status,
		&scan.ScanType,
		&scan.Priority,
		&scan.VulnerabilityScan,
		&scan.MalwareScan,
		&scan.LicenseScan,
		&scan.DependencyScan,
		&scan.InitiatedBy,
		&scan.StartedAt,
		&completedAt,
		&duration,
		&errorMessage,
		&metadataJSON,
		&scan.CreatedAt,
		&scan.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		scan.CompletedAt = &completedAt.Time
	}
	if duration.Valid {
		scan.Duration = &duration.Int64
	}
	if errorMessage.Valid {
		scan.ErrorMessage = &errorMessage.String
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &scan.Metadata)
	}

	// Load scan results if available
	if scan.Status == "completed" || scan.Status == "completed_with_errors" {
		results, err := r.GetResultsByScanID(scan.ID)
		if err == nil {
			scan.Results = results
		}
	}

	return scan, nil
}

// GetByUUID retrieves a scan by its UUID
func (r *ScanRepository) GetByUUID(scanID uuid.UUID) (*models.SecurityScan, error) {
	query := `
		SELECT scan_id, artifact_id, status, scan_type, priority,
			   vulnerability_scan, malware_scan, license_scan, dependency_scan,
			   initiated_by, started_at, completed_at, duration, error_message,
			   metadata, created_at, updated_at
		FROM security_scans
		WHERE scan_id = $1`

	scan := &models.SecurityScan{}
	var metadataJSON []byte
	var completedAt sql.NullTime
	var duration sql.NullInt64
	var errorMessage sql.NullString

	err := r.db.QueryRow(query, scanID).Scan(
		&scan.ID,
		&scan.ArtifactID,
		&scan.Status,
		&scan.ScanType,
		&scan.Priority,
		&scan.VulnerabilityScan,
		&scan.MalwareScan,
		&scan.LicenseScan,
		&scan.DependencyScan,
		&scan.InitiatedBy,
		&scan.StartedAt,
		&completedAt,
		&duration,
		&errorMessage,
		&metadataJSON,
		&scan.CreatedAt,
		&scan.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		scan.CompletedAt = &completedAt.Time
	}
	if duration.Valid {
		scan.Duration = &duration.Int64
	}
	if errorMessage.Valid {
		scan.ErrorMessage = &errorMessage.String
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &scan.Metadata)
	}

	// Load scan results if available
	if scan.Status == "completed" || scan.Status == "completed_with_errors" {
		results, err := r.GetResultsByScanID(scan.ID)
		if err == nil {
			scan.Results = results
		}
	}

	return scan, nil
}

// GetActiveScan retrieves active scan for an artifact
func (r *ScanRepository) GetActiveScan(artifactID uuid.UUID) (*models.SecurityScan, error) {
	query := `
		SELECT scan_id, artifact_id, status, scan_type, priority,
			   vulnerability_scan, malware_scan, license_scan, dependency_scan,
			   initiated_by, started_at, completed_at, duration, error_message,
			   metadata, created_at, updated_at
		FROM security_scans
		WHERE artifact_id = $1 AND status IN ('initiated', 'running')
		ORDER BY started_at DESC
		LIMIT 1`

	scan := &models.SecurityScan{}
	var metadataJSON []byte
	var completedAt sql.NullTime
	var duration sql.NullInt64
	var errorMessage sql.NullString

	err := r.db.QueryRow(query, artifactID).Scan(
		&scan.ID,
		&scan.ArtifactID,
		&scan.Status,
		&scan.ScanType,
		&scan.Priority,
		&scan.VulnerabilityScan,
		&scan.MalwareScan,
		&scan.LicenseScan,
		&scan.DependencyScan,
		&scan.InitiatedBy,
		&scan.StartedAt,
		&completedAt,
		&duration,
		&errorMessage,
		&metadataJSON,
		&scan.CreatedAt,
		&scan.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		scan.CompletedAt = &completedAt.Time
	}
	if duration.Valid {
		scan.Duration = &duration.Int64
	}
	if errorMessage.Valid {
		scan.ErrorMessage = &errorMessage.String
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &scan.Metadata)
	}

	return scan, nil
}

// GetLatestResult retrieves the latest scan result for an artifact
func (r *ScanRepository) GetLatestResult(artifactID uuid.UUID) (*models.SecurityScan, error) {
	query := `
		SELECT scan_id, artifact_id, status, scan_type, priority,
			   vulnerability_scan, malware_scan, license_scan, dependency_scan,
			   initiated_by, started_at, completed_at, duration, error_message,
			   metadata, created_at, updated_at
		FROM security_scans
		WHERE artifact_id = $1 AND status IN ('completed', 'completed_with_errors')
		ORDER BY completed_at DESC
		LIMIT 1`

	scan := &models.SecurityScan{}
	var metadataJSON []byte
	var completedAt sql.NullTime
	var duration sql.NullInt64
	var errorMessage sql.NullString

	err := r.db.QueryRow(query, artifactID).Scan(
		&scan.ID,
		&scan.ArtifactID,
		&scan.Status,
		&scan.ScanType,
		&scan.Priority,
		&scan.VulnerabilityScan,
		&scan.MalwareScan,
		&scan.LicenseScan,
		&scan.DependencyScan,
		&scan.InitiatedBy,
		&scan.StartedAt,
		&completedAt,
		&duration,
		&errorMessage,
		&metadataJSON,
		&scan.CreatedAt,
		&scan.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		scan.CompletedAt = &completedAt.Time
	}
	if duration.Valid {
		scan.Duration = &duration.Int64
	}
	if errorMessage.Valid {
		scan.ErrorMessage = &errorMessage.String
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &scan.Metadata)
	}

	// Load scan results
	results, err := r.GetResultsByScanID(scan.ID)
	if err == nil {
		scan.Results = results
	}

	return scan, nil
}

// GetHistory retrieves scan history for an artifact
func (r *ScanRepository) GetHistory(artifactID uuid.UUID, limit, offset int) ([]*models.SecurityScan, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM security_scans WHERE artifact_id = $1`
	err := r.db.QueryRow(countQuery, artifactID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get scans
	query := `
		SELECT scan_id, artifact_id, status, scan_type, priority,
			   vulnerability_scan, malware_scan, license_scan, dependency_scan,
			   initiated_by, started_at, completed_at, duration, error_message,
			   metadata, created_at, updated_at
		FROM security_scans
		WHERE artifact_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(query, artifactID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var scans []*models.SecurityScan
	for rows.Next() {
		scan := &models.SecurityScan{}
		var metadataJSON []byte
		var completedAt sql.NullTime
		var duration sql.NullInt64
		var errorMessage sql.NullString

		err := rows.Scan(
			&scan.ID,
			&scan.ArtifactID,
			&scan.Status,
			&scan.ScanType,
			&scan.Priority,
			&scan.VulnerabilityScan,
			&scan.MalwareScan,
			&scan.LicenseScan,
			&scan.DependencyScan,
			&scan.InitiatedBy,
			&scan.StartedAt,
			&completedAt,
			&duration,
			&errorMessage,
			&metadataJSON,
			&scan.CreatedAt,
			&scan.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if completedAt.Valid {
			scan.CompletedAt = &completedAt.Time
		}
		if duration.Valid {
			scan.Duration = &duration.Int64
		}
		if errorMessage.Valid {
			scan.ErrorMessage = &errorMessage.String
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &scan.Metadata)
		}

		scans = append(scans, scan)
	}

	return scans, total, nil
}

// GetAllWithFilters retrieves all scans with optional filtering
func (r *ScanRepository) GetAllWithFilters(status, scanType, priority string, limit, offset int) ([]*models.SecurityScan, int, error) {
	baseQuery := `
		FROM security_scans 
		WHERE 1=1`

	var args []interface{}
	argIndex := 1

	// Add filters
	if status != "" && status != "all" {
		baseQuery += ` AND status = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if scanType != "" && scanType != "all" {
		baseQuery += ` AND scan_type = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, scanType)
		argIndex++
	}

	if priority != "" && priority != "all" {
		baseQuery += ` AND priority = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, priority)
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	selectQuery := `
		SELECT scan_id, artifact_id, status, scan_type, priority,
			   vulnerability_scan, malware_scan, license_scan, dependency_scan,
			   initiated_by, started_at, completed_at, duration, error_message,
			   metadata, created_at, updated_at ` +
		baseQuery +
		` ORDER BY started_at DESC LIMIT $` + fmt.Sprintf("%d", argIndex) +
		` OFFSET $` + fmt.Sprintf("%d", argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var scans []*models.SecurityScan
	for rows.Next() {
		scan := &models.SecurityScan{}
		var metadataJSON []byte
		var completedAt sql.NullTime
		var duration sql.NullInt64
		var errorMessage sql.NullString

		err := rows.Scan(
			&scan.ID,
			&scan.ArtifactID,
			&scan.Status,
			&scan.ScanType,
			&scan.Priority,
			&scan.VulnerabilityScan,
			&scan.MalwareScan,
			&scan.LicenseScan,
			&scan.DependencyScan,
			&scan.InitiatedBy,
			&scan.StartedAt,
			&completedAt,
			&duration,
			&errorMessage,
			&metadataJSON,
			&scan.CreatedAt,
			&scan.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if completedAt.Valid {
			scan.CompletedAt = &completedAt.Time
		}
		if duration.Valid {
			scan.Duration = &duration.Int64
		}
		if errorMessage.Valid {
			scan.ErrorMessage = &errorMessage.String
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &scan.Metadata)
		}

		scans = append(scans, scan)
	}

	return scans, total, nil
}

// SaveResults saves scan results
func (r *ScanRepository) SaveResults(results *models.ScanResults) error {
	query := `
		INSERT INTO scan_results (
			scan_id, tenant_id, overall_score, risk_level, summary, recommendations,
			vulnerability_results, malware_results, license_results, dependency_results,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING result_id`

	vulnResultsJSON, _ := json.Marshal(results.VulnerabilityResults)
	malwareResultsJSON, _ := json.Marshal(results.MalwareResults)
	licenseResultsJSON, _ := json.Marshal(results.LicenseResults)
	depResultsJSON, _ := json.Marshal(results.DependencyResults)
	recommendationsJSON, _ := json.Marshal(results.Recommendations)

	err := r.db.QueryRow(
		query,
		results.ScanID,
		results.TenantID,
		results.OverallScore,
		results.RiskLevel,
		results.Summary,
		recommendationsJSON,
		vulnResultsJSON,
		malwareResultsJSON,
		licenseResultsJSON,
		depResultsJSON,
		results.CreatedAt,
		results.UpdatedAt,
	).Scan(&results.ID)

	return err
}

// GetResultsByScanID retrieves scan results by scan ID
func (r *ScanRepository) GetResultsByScanID(scanID uuid.UUID) (*models.ScanResults, error) {
	query := `
		SELECT result_id, scan_id, tenant_id, overall_score, risk_level, summary, recommendations,
			   vulnerability_results, malware_results, license_results, dependency_results,
			   created_at, updated_at
		FROM scan_results
		WHERE scan_id = $1`

	results := &models.ScanResults{}
	var vulnResultsJSON, malwareResultsJSON, licenseResultsJSON, depResultsJSON []byte
	var recommendationsJSON []byte

	err := r.db.QueryRow(query, scanID).Scan(
		&results.ID,
		&results.ScanID,
		&results.TenantID,
		&results.OverallScore,
		&results.RiskLevel,
		&results.Summary,
		&recommendationsJSON,
		&vulnResultsJSON,
		&malwareResultsJSON,
		&licenseResultsJSON,
		&depResultsJSON,
		&results.CreatedAt,
		&results.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(recommendationsJSON) > 0 {
		json.Unmarshal(recommendationsJSON, &results.Recommendations)
	}

	if len(vulnResultsJSON) > 0 {
		var vulnResults models.VulnerabilityResults
		if json.Unmarshal(vulnResultsJSON, &vulnResults) == nil {
			results.VulnerabilityResults = &vulnResults
		}
	}

	if len(malwareResultsJSON) > 0 {
		var malwareResults models.MalwareResults
		if json.Unmarshal(malwareResultsJSON, &malwareResults) == nil {
			results.MalwareResults = &malwareResults
		}
	}

	if len(licenseResultsJSON) > 0 {
		var licenseResults models.LicenseResults
		if json.Unmarshal(licenseResultsJSON, &licenseResults) == nil {
			results.LicenseResults = &licenseResults
		}
	}

	if len(depResultsJSON) > 0 {
		var depResults models.DependencyResults
		if json.Unmarshal(depResultsJSON, &depResults) == nil {
			results.DependencyResults = &depResults
		}
	}

	return results, nil
}

// GetStatistics returns scan statistics
func (r *ScanRepository) GetStatistics() (*models.ScanStatistics, error) {
	stats := &models.ScanStatistics{
		ScansByType:     make(map[string]int64),
		ScansByPriority: make(map[string]int64),
	}

	// Get basic counts
	basicQuery := `
		SELECT 
			COUNT(*) as total_scans,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_scans,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_scans,
			COUNT(CASE WHEN status IN ('initiated', 'running') THEN 1 END) as active_scans,
			AVG(CASE WHEN duration IS NOT NULL THEN duration / 60.0 END) as avg_time_minutes
		FROM security_scans`

	err := r.db.QueryRow(basicQuery).Scan(
		&stats.TotalScans,
		&stats.CompletedScans,
		&stats.FailedScans,
		&stats.ActiveScans,
		&stats.AverageTime,
	)
	if err != nil {
		return nil, err
	}

	// Get scans by type
	typeQuery := `SELECT scan_type, COUNT(*) FROM security_scans GROUP BY scan_type`
	rows, err := r.db.Query(typeQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var scanType string
		var count int64
		if err := rows.Scan(&scanType, &count); err == nil {
			stats.ScansByType[scanType] = count
		}
	}

	// Get scans by priority
	priorityQuery := `SELECT priority, COUNT(*) FROM security_scans GROUP BY priority`
	rows, err = r.db.Query(priorityQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var priority string
		var count int64
		if err := rows.Scan(&priority, &count); err == nil {
			stats.ScansByPriority[priority] = count
		}
	}

	// Get vulnerability and threat counts from latest scans
	vulnQuery := `
		SELECT 
			COALESCE(SUM(
				CAST(vulnerability_results->>'total_found' AS INTEGER)
			), 0) as total_vulns,
		COALESCE(SUM(
			CAST(malware_results->>'threats_found' AS INTEGER)
		), 0) as total_threats
	FROM scan_results sr
	JOIN security_scans ss ON sr.scan_id = ss.scan_id
	WHERE ss.status = 'completed'`

	err = r.db.QueryRow(vulnQuery).Scan(
		&stats.VulnerabilitiesFound,
		&stats.ThreatsDetected,
	)
	if err != nil {
		// Don't fail on this error, just set to 0
		stats.VulnerabilitiesFound = 0
		stats.ThreatsDetected = 0
	}

	return stats, nil
}

// Tenant-aware methods for multi-tenant isolation

func (r *ScanRepository) GetAllWithFiltersByTenant(tenantID uuid.UUID, status, scanType, priority string, limit, offset int) ([]*models.SecurityScan, int, error) {
	baseQuery := `
		FROM security_scans ss
		LEFT JOIN artifacts a ON ss.artifact_id = a.artifact_id
		WHERE ss.tenant_id = $1`

	args := []interface{}{tenantID}
	argIndex := 2

	// Add filters
	if status != "" && status != "all" {
		baseQuery += ` AND ss.status = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if scanType != "" && scanType != "all" {
		baseQuery += ` AND ss.scan_type = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, scanType)
		argIndex++
	}

	if priority != "" && priority != "all" {
		baseQuery += ` AND ss.priority = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, priority)
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(DISTINCT ss.scan_id) " + baseQuery
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count scans: %w", err)
	}

	// Get scans with pagination - include metadata for artifact info
	query := `SELECT 
		ss.scan_id, ss.artifact_id, ss.status, ss.scan_type, ss.priority,
		ss.vulnerability_scan, ss.malware_scan, ss.license_scan, ss.dependency_scan,
		ss.initiated_by, ss.started_at, ss.completed_at, ss.duration, ss.error_message,
		ss.metadata, ss.created_at, ss.updated_at,
		a.name, a.version
		` + baseQuery + `
		ORDER BY ss.started_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query scans: %w", err)
	}
	defer rows.Close()

	scans := make([]*models.SecurityScan, 0)
	for rows.Next() {
		scan := &models.SecurityScan{}
		var metadataJSON sql.NullString
		var artifactName, version sql.NullString
		var completedAt sql.NullTime
		var duration sql.NullInt64
		var errorMessage sql.NullString

		err := rows.Scan(
			&scan.ID,
			&scan.ArtifactID,
			&scan.Status,
			&scan.ScanType,
			&scan.Priority,
			&scan.VulnerabilityScan,
			&scan.MalwareScan,
			&scan.LicenseScan,
			&scan.DependencyScan,
			&scan.InitiatedBy,
			&scan.StartedAt,
			&completedAt,
			&duration,
			&errorMessage,
			&metadataJSON,
			&scan.CreatedAt,
			&scan.UpdatedAt,
			&artifactName,
			&version,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert nullable database fields to pointers
		if completedAt.Valid {
			scan.CompletedAt = &completedAt.Time
		}
		if duration.Valid {
			scan.Duration = &duration.Int64
		}
		if errorMessage.Valid {
			scan.ErrorMessage = &errorMessage.String
		}

		// Initialize metadata map first
		scan.Metadata = make(map[string]interface{})

		// Parse metadata JSON if present and merge
		if metadataJSON.Valid && metadataJSON.String != "" {
			var metadata map[string]interface{}
			err = json.Unmarshal([]byte(metadataJSON.String), &metadata)
			if err == nil && metadata != nil {
				// Copy parsed metadata into scan.Metadata
				for k, v := range metadata {
					scan.Metadata[k] = v
				}
			}
		}

		// Add artifact info from artifacts table if available (for non-cache scans)
		// This will override metadata if both exist
		if artifactName.Valid {
			scan.Metadata["artifact_name"] = artifactName.String
		}
		if version.Valid {
			scan.Metadata["version"] = version.String
		}

		scans = append(scans, scan)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating over scan rows: %w", err)
	}

	return scans, total, nil
}

func (r *ScanRepository) GetLatestResultByTenant(artifactID, tenantID uuid.UUID) (*models.SecurityScan, error) {
	query := `
		SELECT ss.scan_id, ss.artifact_id, ss.status, ss.scan_type, ss.priority,
			ss.vulnerability_scan, ss.malware_scan, ss.license_scan, ss.dependency_scan,
			ss.initiated_by, ss.started_at, ss.completed_at, ss.duration, ss.error_message,
			ss.created_at, ss.updated_at
		FROM security_scans ss
		JOIN artifacts a ON ss.artifact_id = a.artifact_id
		WHERE ss.artifact_id = $1 AND a.tenant_id = $2 AND ss.status = 'completed'
		ORDER BY ss.completed_at DESC
		LIMIT 1`

	scan := &models.SecurityScan{}
	err := r.db.QueryRow(query, artifactID, tenantID).Scan(
		&scan.ID,
		&scan.ArtifactID,
		&scan.Status,
		&scan.ScanType,
		&scan.Priority,
		&scan.VulnerabilityScan,
		&scan.MalwareScan,
		&scan.LicenseScan,
		&scan.DependencyScan,
		&scan.InitiatedBy,
		&scan.StartedAt,
		&scan.CompletedAt,
		&scan.Duration,
		&scan.ErrorMessage,
		&scan.CreatedAt,
		&scan.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no completed scan found for this artifact")
		}
		return nil, fmt.Errorf("failed to get latest scan: %w", err)
	}

	// Load scan results
	scan.Results, _ = r.GetResultsByScanID(scan.ID)

	return scan, nil
}
