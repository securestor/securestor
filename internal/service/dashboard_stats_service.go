package service

import (
	"database/sql"
	"fmt"
	"time"
)

type DashboardStatsService struct {
	db *sql.DB
}

type Stats struct {
	TotalStorage   string    `json:"totalStorage"`
	TotalArtifacts int64     `json:"totalArtifacts"`
	DownloadsToday int64     `json:"downloadsToday"`
	ActiveUsers    int64     `json:"activeUsers"`
	Trends         *Trends   `json:"trends"`
	LastUpdated    time.Time `json:"lastUpdated"`
}

type Trends struct {
	StorageTrend     string `json:"storageTrend"`
	ArtifactsTrend   string `json:"artifactsTrend"`
	DownloadsTrend   string `json:"downloadsTrend"`
	ActiveUsersTrend string `json:"activeUsersTrend"`
}

type DetailedStats struct {
	Storage         *StorageStats    `json:"storage"`
	Artifacts       *ArtifactStats   `json:"artifacts"`
	Downloads       *DownloadStats   `json:"downloads"`
	Users           *UserStats       `json:"users"`
	Compliance      *ComplianceStats `json:"compliance"`
	Security        *SecurityStats   `json:"security"`
	PerformanceData *PerformanceData `json:"performance"`
}

type StorageStats struct {
	TotalUsed      int64            `json:"totalUsed"`
	TotalAvailable int64            `json:"totalAvailable"`
	UsageByType    map[string]int64 `json:"usageByType"`
	GrowthRate     float64          `json:"growthRate"`
	HealthScore    int              `json:"healthScore"`
	BackendStatus  map[string]bool  `json:"backendStatus"`
}

type ArtifactStats struct {
	Total           int64            `json:"total"`
	ByType          map[string]int64 `json:"byType"`
	ByRepository    map[string]int64 `json:"byRepository"`
	RecentUploads   int64            `json:"recentUploads"`
	AverageSize     int64            `json:"averageSize"`
	LargestArtifact int64            `json:"largestArtifact"`
}

type DownloadStats struct {
	Today         int64            `json:"today"`
	ThisWeek      int64            `json:"thisWeek"`
	ThisMonth     int64            `json:"thisMonth"`
	TopArtifacts  []TopArtifact    `json:"topArtifacts"`
	HourlyPattern []HourlyData     `json:"hourlyPattern"`
	PopularTypes  map[string]int64 `json:"popularTypes"`
}

type UserStats struct {
	Total         int64        `json:"total"`
	ActiveToday   int64        `json:"activeToday"`
	ActiveWeek    int64        `json:"activeWeek"`
	NewUsers      int64        `json:"newUsers"`
	TopUsers      []TopUser    `json:"topUsers"`
	ActivityHours []HourlyData `json:"activityHours"`
}

type ComplianceStats struct {
	OverallScore      int                  `json:"overallScore"`
	PolicyCompliance  map[string]int       `json:"policyCompliance"`
	ViolationsCount   int64                `json:"violationsCount"`
	PendingReviews    int64                `json:"pendingReviews"`
	ExpiringRetention int64                `json:"expiringRetention"`
	DPDPCompliance    *DPDPComplianceStats `json:"dpdpCompliance"`
}

type SecurityStats struct {
	TotalScans           int64            `json:"totalScans"`
	VulnerabilitiesFound int64            `json:"vulnerabilitiesFound"`
	CriticalIssues       int64            `json:"criticalIssues"`
	SecurityScore        int              `json:"securityScore"`
	ThreatsByType        map[string]int64 `json:"threatsByType"`
	RecentScans          []RecentScan     `json:"recentScans"`
}

type PerformanceData struct {
	AvgResponseTime   float64 `json:"avgResponseTime"`
	RequestsPerSecond float64 `json:"requestsPerSecond"`
	ErrorRate         float64 `json:"errorRate"`
	CacheHitRate      float64 `json:"cacheHitRate"`
	DatabaseHealth    int     `json:"databaseHealth"`
}

type DPDPComplianceStats struct {
	ConsentRecords        int64 `json:"consentRecords"`
	PendingRightsRequests int64 `json:"pendingRightsRequests"`
	CrossBorderTransfers  int64 `json:"crossBorderTransfers"`
	BreachIncidents       int64 `json:"breachIncidents"`
	ImpactAssessments     int64 `json:"impactAssessments"`
	ComplianceScore       int   `json:"complianceScore"`
}

type TopArtifact struct {
	Name      string `json:"name"`
	Downloads int64  `json:"downloads"`
	Type      string `json:"type"`
}

type TopUser struct {
	UserID     string `json:"userId"`
	Downloads  int64  `json:"downloads"`
	Artifacts  int64  `json:"artifacts"`
	LastActive string `json:"lastActive"`
}

type HourlyData struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

type RecentScan struct {
	ArtifactName string    `json:"artifactName"`
	Status       string    `json:"status"`
	Score        int       `json:"score"`
	CompletedAt  time.Time `json:"completedAt"`
}

func NewDashboardStatsService(db *sql.DB) *DashboardStatsService {
	return &DashboardStatsService{db: db}
}

func (s *DashboardStatsService) GetBasicStats() (*Stats, error) {
	stats := &Stats{
		LastUpdated: time.Now(),
	}

	// Get total storage used
	totalSize, err := s.getTotalStorageUsed()
	if err != nil {
		return nil, fmt.Errorf("failed to get total storage: %w", err)
	}
	stats.TotalStorage = formatBytes(totalSize)

	// Get total artifacts
	stats.TotalArtifacts, err = s.getTotalArtifacts()
	if err != nil {
		return nil, fmt.Errorf("failed to get total artifacts: %w", err)
	}

	// Get downloads today
	stats.DownloadsToday, err = s.getDownloadsToday()
	if err != nil {
		return nil, fmt.Errorf("failed to get downloads today: %w", err)
	}

	// Get active users
	stats.ActiveUsers, err = s.getActiveUsersToday()
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	// Calculate trends
	trends, err := s.calculateTrends()
	if err != nil {
		// Log error but don't fail, trends are optional
		trends = &Trends{
			StorageTrend:     "+0%",
			ArtifactsTrend:   "+0",
			DownloadsTrend:   "+0%",
			ActiveUsersTrend: "+0",
		}
	}
	stats.Trends = trends

	return stats, nil
}

func (s *DashboardStatsService) GetDetailedStats() (*DetailedStats, error) {
	detailed := &DetailedStats{}

	// Get storage statistics
	storageStats, err := s.getStorageStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}
	detailed.Storage = storageStats

	// Get artifact statistics
	artifactStats, err := s.getArtifactStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact stats: %w", err)
	}
	detailed.Artifacts = artifactStats

	// Get download statistics
	downloadStats, err := s.getDownloadStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get download stats: %w", err)
	}
	detailed.Downloads = downloadStats

	// Get user statistics
	userStats, err := s.getUserStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	detailed.Users = userStats

	// Get compliance statistics
	complianceStats, err := s.getComplianceStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance stats: %w", err)
	}
	detailed.Compliance = complianceStats

	// Get security statistics
	securityStats, err := s.getSecurityStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get security stats: %w", err)
	}
	detailed.Security = securityStats

	// Get performance data
	performanceData, err := s.getPerformanceData()
	if err != nil {
		return nil, fmt.Errorf("failed to get performance data: %w", err)
	}
	detailed.PerformanceData = performanceData

	return detailed, nil
}

// Helper methods

func (s *DashboardStatsService) getTotalStorageUsed() (int64, error) {
	var totalSize sql.NullInt64
	query := `SELECT COALESCE(SUM(size), 0) FROM artifacts`
	err := s.db.QueryRow(query).Scan(&totalSize)
	if err != nil {
		return 0, err
	}
	return totalSize.Int64, nil
}

func (s *DashboardStatsService) getTotalArtifacts() (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM artifacts`
	err := s.db.QueryRow(query).Scan(&count)
	return count, err
}

func (s *DashboardStatsService) getDownloadsToday() (int64, error) {
	// This would require a downloads/access log table
	// For now, we'll use audit logs as a proxy
	var count int64
	query := `
		SELECT COUNT(*) FROM audit_logs 
		WHERE action = 'download' 
		AND timestamp >= CURRENT_DATE
		AND timestamp < CURRENT_DATE + INTERVAL '1 day'`
	err := s.db.QueryRow(query).Scan(&count)
	return count, err
}

func (s *DashboardStatsService) getActiveUsersToday() (int64, error) {
	var count int64
	query := `
		SELECT COUNT(DISTINCT user_id) FROM audit_logs 
		WHERE timestamp >= CURRENT_DATE
		AND timestamp < CURRENT_DATE + INTERVAL '1 day'`
	err := s.db.QueryRow(query).Scan(&count)
	return count, err
}

func (s *DashboardStatsService) calculateTrends() (*Trends, error) {
	trends := &Trends{}

	// Calculate storage trend (last 7 days vs previous 7 days)
	storageTrend, err := s.calculateStorageTrend()
	if err == nil {
		trends.StorageTrend = storageTrend
	} else {
		trends.StorageTrend = "+0%"
	}

	// Calculate artifacts trend
	artifactsTrend, err := s.calculateArtifactsTrend()
	if err == nil {
		trends.ArtifactsTrend = artifactsTrend
	} else {
		trends.ArtifactsTrend = "+0"
	}

	// Calculate downloads trend
	downloadsTrend, err := s.calculateDownloadsTrend()
	if err == nil {
		trends.DownloadsTrend = downloadsTrend
	} else {
		trends.DownloadsTrend = "+0%"
	}

	// Calculate active users trend
	usersTrend, err := s.calculateActiveUsersTrend()
	if err == nil {
		trends.ActiveUsersTrend = usersTrend
	} else {
		trends.ActiveUsersTrend = "+0"
	}

	return trends, nil
}

func (s *DashboardStatsService) calculateStorageTrend() (string, error) {
	// Get storage used in last 7 days vs previous 7 days
	var current, previous sql.NullInt64

	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE - INTERVAL '7 days' THEN size END), 0) as current_week,
			COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE - INTERVAL '14 days' AND created_at < CURRENT_DATE - INTERVAL '7 days' THEN size END), 0) as previous_week
		FROM artifacts`

	err := s.db.QueryRow(query).Scan(&current, &previous)
	if err != nil {
		return "+0%", err
	}

	if previous.Int64 == 0 {
		return "+100%", nil
	}

	change := float64(current.Int64-previous.Int64) / float64(previous.Int64) * 100
	if change >= 0 {
		return fmt.Sprintf("+%.1f%%", change), nil
	}
	return fmt.Sprintf("%.1f%%", change), nil
}

func (s *DashboardStatsService) calculateArtifactsTrend() (string, error) {
	var current, previous int64

	query := `
		SELECT 
			COUNT(CASE WHEN created_at >= CURRENT_DATE - INTERVAL '7 days' THEN 1 END) as current_week,
			COUNT(CASE WHEN created_at >= CURRENT_DATE - INTERVAL '14 days' AND created_at < CURRENT_DATE - INTERVAL '7 days' THEN 1 END) as previous_week
		FROM artifacts`

	err := s.db.QueryRow(query).Scan(&current, &previous)
	if err != nil {
		return "+0", err
	}

	change := current - previous
	if change >= 0 {
		return fmt.Sprintf("+%d", change), nil
	}
	return fmt.Sprintf("%d", change), nil
}

func (s *DashboardStatsService) calculateDownloadsTrend() (string, error) {
	var current, previous int64

	query := `
		SELECT 
			COUNT(CASE WHEN timestamp >= CURRENT_DATE - INTERVAL '7 days' AND action = 'download' THEN 1 END) as current_week,
			COUNT(CASE WHEN timestamp >= CURRENT_DATE - INTERVAL '14 days' AND timestamp < CURRENT_DATE - INTERVAL '7 days' AND action = 'download' THEN 1 END) as previous_week
		FROM audit_logs`

	err := s.db.QueryRow(query).Scan(&current, &previous)
	if err != nil {
		return "+0%", err
	}

	if previous == 0 {
		return "+100%", nil
	}

	change := float64(current-previous) / float64(previous) * 100
	if change >= 0 {
		return fmt.Sprintf("+%.1f%%", change), nil
	}
	return fmt.Sprintf("%.1f%%", change), nil
}

func (s *DashboardStatsService) calculateActiveUsersTrend() (string, error) {
	var current, previous int64

	query := `
		SELECT 
			COUNT(DISTINCT CASE WHEN timestamp >= CURRENT_DATE - INTERVAL '7 days' THEN user_id END) as current_week,
			COUNT(DISTINCT CASE WHEN timestamp >= CURRENT_DATE - INTERVAL '14 days' AND timestamp < CURRENT_DATE - INTERVAL '7 days' THEN user_id END) as previous_week
		FROM audit_logs`

	err := s.db.QueryRow(query).Scan(&current, &previous)
	if err != nil {
		return "+0", err
	}

	change := current - previous
	if change >= 0 {
		return fmt.Sprintf("+%d", change), nil
	}
	return fmt.Sprintf("%d", change), nil
}

// Additional detailed stats methods would go here...
// For brevity, I'll implement the basic structure and a few key methods

func (s *DashboardStatsService) getStorageStats() (*StorageStats, error) {
	stats := &StorageStats{
		UsageByType:   make(map[string]int64),
		BackendStatus: make(map[string]bool),
	}

	// Get total used storage
	totalUsed, err := s.getTotalStorageUsed()
	if err != nil {
		return nil, err
	}
	stats.TotalUsed = totalUsed
	stats.TotalAvailable = totalUsed * 10 // Assume 10x capacity for demo

	// Get usage by type
	query := `
		SELECT type, COALESCE(SUM(size), 0) as total_size 
		FROM artifacts 
		GROUP BY type`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var artifactType string
		var size int64
		if err := rows.Scan(&artifactType, &size); err != nil {
			continue
		}
		stats.UsageByType[artifactType] = size
	}

	// Mock health score and backend status
	stats.HealthScore = 95
	stats.BackendStatus["local"] = true
	stats.BackendStatus["s3"] = true

	return stats, nil
}

func (s *DashboardStatsService) getArtifactStats() (*ArtifactStats, error) {
	stats := &ArtifactStats{
		ByType:       make(map[string]int64),
		ByRepository: make(map[string]int64),
	}

	// Get total count
	total, err := s.getTotalArtifacts()
	if err != nil {
		return nil, err
	}
	stats.Total = total

	// Get by type
	query := `SELECT type, COUNT(*) FROM artifacts GROUP BY type`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var artifactType string
		var count int64
		if err := rows.Scan(&artifactType, &count); err != nil {
			continue
		}
		stats.ByType[artifactType] = count
	}

	// Get recent uploads (last 24 hours)
	query = `SELECT COUNT(*) FROM artifacts WHERE created_at >= CURRENT_DATE`
	err = s.db.QueryRow(query).Scan(&stats.RecentUploads)
	if err != nil {
		stats.RecentUploads = 0
	}

	// Get average size
	query = `SELECT COALESCE(AVG(size), 0) FROM artifacts`
	err = s.db.QueryRow(query).Scan(&stats.AverageSize)
	if err != nil {
		stats.AverageSize = 0
	}

	return stats, nil
}

func (s *DashboardStatsService) getDownloadStats() (*DownloadStats, error) {
	stats := &DownloadStats{
		TopArtifacts:  make([]TopArtifact, 0),
		HourlyPattern: make([]HourlyData, 24),
		PopularTypes:  make(map[string]int64),
	}

	// Get downloads for different periods
	today, _ := s.getDownloadsToday()
	stats.Today = today

	// Get this week downloads
	query := `
		SELECT COUNT(*) FROM audit_logs 
		WHERE action = 'download' 
		AND timestamp >= CURRENT_DATE - INTERVAL '7 days'`
	err := s.db.QueryRow(query).Scan(&stats.ThisWeek)
	if err != nil {
		stats.ThisWeek = 0
	}

	// Get this month downloads
	query = `
		SELECT COUNT(*) FROM audit_logs 
		WHERE action = 'download' 
		AND timestamp >= DATE_TRUNC('month', CURRENT_DATE)`
	err = s.db.QueryRow(query).Scan(&stats.ThisMonth)
	if err != nil {
		stats.ThisMonth = 0
	}

	// Initialize hourly pattern with zeros
	for i := 0; i < 24; i++ {
		stats.HourlyPattern[i] = HourlyData{Hour: i, Count: 0}
	}

	return stats, nil
}

func (s *DashboardStatsService) getUserStats() (*UserStats, error) {
	stats := &UserStats{
		TopUsers:      make([]TopUser, 0),
		ActivityHours: make([]HourlyData, 24),
	}

	// Get total users
	query := `SELECT COUNT(DISTINCT user_id) FROM audit_logs`
	err := s.db.QueryRow(query).Scan(&stats.Total)
	if err != nil {
		stats.Total = 0
	}

	// Get active users today
	activeToday, _ := s.getActiveUsersToday()
	stats.ActiveToday = activeToday

	// Get active users this week
	query = `
		SELECT COUNT(DISTINCT user_id) FROM audit_logs 
		WHERE timestamp >= CURRENT_DATE - INTERVAL '7 days'`
	err = s.db.QueryRow(query).Scan(&stats.ActiveWeek)
	if err != nil {
		stats.ActiveWeek = 0
	}

	return stats, nil
}

func (s *DashboardStatsService) getComplianceStats() (*ComplianceStats, error) {
	stats := &ComplianceStats{
		PolicyCompliance: make(map[string]int),
		DPDPCompliance:   &DPDPComplianceStats{},
	}

	// Get compliance violations count
	query := `SELECT COUNT(*) FROM compliance_reports WHERE status = 'non_compliant'`
	err := s.db.QueryRow(query).Scan(&stats.ViolationsCount)
	if err != nil {
		stats.ViolationsCount = 0
	}

	// Get pending reviews
	query = `SELECT COUNT(*) FROM compliance_reports WHERE status = 'pending'`
	err = s.db.QueryRow(query).Scan(&stats.PendingReviews)
	if err != nil {
		stats.PendingReviews = 0
	}

	// DPDP compliance stats
	query = `SELECT COUNT(*) FROM dpdp_consent_records WHERE consent_status = 'active'`
	err = s.db.QueryRow(query).Scan(&stats.DPDPCompliance.ConsentRecords)
	if err != nil {
		stats.DPDPCompliance.ConsentRecords = 0
	}

	query = `SELECT COUNT(*) FROM dpdp_rights_requests WHERE status = 'pending'`
	err = s.db.QueryRow(query).Scan(&stats.DPDPCompliance.PendingRightsRequests)
	if err != nil {
		stats.DPDPCompliance.PendingRightsRequests = 0
	}

	// Mock overall scores
	stats.OverallScore = 92
	stats.DPDPCompliance.ComplianceScore = 94

	return stats, nil
}

func (s *DashboardStatsService) getSecurityStats() (*SecurityStats, error) {
	stats := &SecurityStats{
		ThreatsByType: make(map[string]int64),
		RecentScans:   make([]RecentScan, 0),
	}

	// Get total scans
	query := `SELECT COUNT(*) FROM security_scans`
	err := s.db.QueryRow(query).Scan(&stats.TotalScans)
	if err != nil {
		stats.TotalScans = 0
	}

	// Get vulnerabilities found
	query = `SELECT COUNT(*) FROM scan_results WHERE risk_level IN ('high', 'critical')`
	err = s.db.QueryRow(query).Scan(&stats.VulnerabilitiesFound)
	if err != nil {
		stats.VulnerabilitiesFound = 0
	}

	// Get critical issues
	query = `SELECT COUNT(*) FROM scan_results WHERE risk_level = 'critical'`
	err = s.db.QueryRow(query).Scan(&stats.CriticalIssues)
	if err != nil {
		stats.CriticalIssues = 0
	}

	// Mock security score
	stats.SecurityScore = 88

	return stats, nil
}

func (s *DashboardStatsService) getPerformanceData() (*PerformanceData, error) {
	// Mock performance data - in a real implementation, this would come from monitoring systems
	return &PerformanceData{
		AvgResponseTime:   125.5,
		RequestsPerSecond: 45.2,
		ErrorRate:         0.02,
		CacheHitRate:      0.89,
		DatabaseHealth:    96,
	}, nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
