package failover

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/securestor/securestor/internal/logger"
)

// PatroniService manages Patroni cluster coordination
type PatroniService struct {
	apiURL      string
	clusterName string
	nodeName    string
	client      *http.Client
	logger      *logger.Logger
	mutex       sync.RWMutex
	lastStatus  *PatroniStatus
}

// PatroniStatus represents the response from Patroni API
type PatroniStatus struct {
	State             string             `json:"state"`
	Role              string             `json:"role"`
	Xlog              PatroniXlog        `json:"xlog"`
	ClusterInfo       PatroniClusterInfo `json:"cluster_info"`
	Members           []PatroniMember    `json:"members"`
	LastHeartbeatTime string             `json:"last_heartbeat_time"`
	Bootstrap         bool               `json:"bootstrap"`
}

// PatroniXlog contains transaction log info
type PatroniXlog struct {
	DatacheckSum string `json:"data_checksum"`
}

// PatroniClusterInfo contains cluster metadata
type PatroniClusterInfo struct {
	TtlSecs      int    `json:"ttl_secs"`
	Scope        string `json:"scope"`
	LastTimeline int    `json:"last_timeline"`
}

// PatroniMember represents a cluster member
type PatroniMember struct {
	Name          string `json:"name"`
	Role          string `json:"role"`
	Index         int    `json:"index"`
	State         string `json:"state"`
	Conn_url      string `json:"conn_url"`
	ApiUrl        string `json:"api_url"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Lag           int64  `json:"lag"`
	Xlog_location string `json:"xlog_location"`
}

// PatroniMembersResponse is the response from /cluster endpoint
type PatroniMembersResponse struct {
	Members []PatroniMember `json:"members"`
}

// NewPatroniService creates a new Patroni service
func NewPatroniService(apiURL, clusterName, nodeName string, logger *logger.Logger) *PatroniService {
	return &PatroniService{
		apiURL:      apiURL,
		clusterName: clusterName,
		nodeName:    nodeName,
		client:      &http.Client{Timeout: 5 * time.Second},
		logger:      logger,
	}
}

// GetNodeStatus gets the current node status from Patroni
func (ps *PatroniService) GetNodeStatus(ctx context.Context) (*PatroniStatus, error) {
	url := fmt.Sprintf("%s/status", ps.apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ps.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("patroni status returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var status PatroniStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, err
	}

	ps.mutex.Lock()
	ps.lastStatus = &status
	ps.mutex.Unlock()

	return &status, nil
}

// GetClusterMembers gets all cluster members from Patroni
func (ps *PatroniService) GetClusterMembers(ctx context.Context) ([]PatroniMember, error) {
	url := fmt.Sprintf("%s/cluster", ps.apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ps.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("patroni cluster returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var membersResp PatroniMembersResponse
	if err := json.Unmarshal(body, &membersResp); err != nil {
		return nil, err
	}

	return membersResp.Members, nil
}

// Failover requests a failover
func (ps *PatroniService) Failover(ctx context.Context, target string) error {
	url := fmt.Sprintf("%s/failover", ps.apiURL)

	payload := map[string]interface{}{
		"leader":    "*",
		"candidate": target,
		"scheduled": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ps.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failover returned %d", resp.StatusCode)
	}

	ps.logger.Printf("✅ Failover to %s requested", target)
	return nil
}

// Switchover requests a controlled switchover
func (ps *PatroniService) Switchover(ctx context.Context, target string, scheduled bool) error {
	url := fmt.Sprintf("%s/switchover", ps.apiURL)

	payload := map[string]interface{}{
		"leader":    "*",
		"candidate": target,
		"scheduled": scheduled,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ps.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("switchover returned %d", resp.StatusCode)
	}

	ps.logger.Printf("✅ Switchover to %s requested", target)
	return nil
}

// GetPrimaryNode returns the current primary node
func (ps *PatroniService) GetPrimaryNode(ctx context.Context) (*PatroniMember, error) {
	members, err := ps.GetClusterMembers(ctx)
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		if member.Role == "master" || member.Role == "primary" {
			return &member, nil
		}
	}

	return nil, fmt.Errorf("no primary found in cluster")
}

// GetHealthyStandbys returns all healthy standby nodes
func (ps *PatroniService) GetHealthyStandbys(ctx context.Context) ([]PatroniMember, error) {
	members, err := ps.GetClusterMembers(ctx)
	if err != nil {
		return nil, err
	}

	healthy := make([]PatroniMember, 0)
	for _, member := range members {
		if (member.Role == "replica" || member.Role == "standby") && member.State == "running" {
			healthy = append(healthy, member)
		}
	}

	return healthy, nil
}

// MonitorCluster continuously monitors cluster health
func (ps *PatroniService) MonitorCluster(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ps.logger.Printf("Cluster monitoring stopped")
			return
		case <-ticker.C:
			status, err := ps.GetNodeStatus(ctx)
			if err != nil {
				ps.logger.Printf("ERROR: Failed to get node status: %v", err)
				continue
			}

			ps.logger.Printf("Node role: %s, state: %s", status.Role, status.State)

			// Check cluster members
			members, err := ps.GetClusterMembers(ctx)
			if err == nil {
				ps.logger.Printf("Cluster members: %d", len(members))
				for _, member := range members {
					ps.logger.Printf("  - %s: %s/%s (lag: %d)",
						member.Name, member.Role, member.State, member.Lag)
				}
			}
		}
	}
}

// GetLastStatus returns the last cached status
func (ps *PatroniService) GetLastStatus() *PatroniStatus {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.lastStatus
}

// IsLeader checks if the current node is the leader
func (ps *PatroniService) IsLeader(ctx context.Context) (bool, error) {
	status, err := ps.GetNodeStatus(ctx)
	if err != nil {
		return false, err
	}

	return status.Role == "master" || status.Role == "primary", nil
}

// GetClusterHealth returns overall cluster health
func (ps *PatroniService) GetClusterHealth(ctx context.Context) (string, error) {
	members, err := ps.GetClusterMembers(ctx)
	if err != nil {
		return "unknown", err
	}

	totalMembers := len(members)
	healthyMembers := 0
	primaryFound := false

	for _, member := range members {
		if member.State == "running" {
			healthyMembers++
		}
		if member.Role == "master" || member.Role == "primary" {
			primaryFound = true
		}
	}

	// Cluster health determination
	if !primaryFound {
		return "critical", fmt.Errorf("no primary found")
	}

	if healthyMembers < (totalMembers / 2) {
		return "degraded", nil
	}

	return "healthy", nil
}
