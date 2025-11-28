package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// SimpleOrchestrator runs a configurable set of scanners in parallel and aggregates results
type SimpleOrchestrator struct {
	scanners       []Scanner
	policy         PolicyClient
	outStore       OutputStore
	workerPoolSize int
}

// OutputStore persists scan results (DB/Blob/FS). Implemented by caller.
type OutputStore interface {
	SaveScanResults(ctx context.Context, jobID string, results []ScannerResult) error
	MarkJobCompleted(ctx context.Context, jobID string, status string) error
}

func NewOrchestrator(scanners []Scanner, policy PolicyClient, out OutputStore, pool int) *SimpleOrchestrator {
	return &SimpleOrchestrator{scanners: scanners, policy: policy, outStore: out, workerPoolSize: pool}
}

func (o *SimpleOrchestrator) Submit(ctx context.Context, job ScanJob) error {
	// Select applicable scanners
	var applicable []Scanner
	for _, s := range o.scanners {
		if s.Supports(job.ArtifactType) {
			applicable = append(applicable, s)
		}
	}
	if len(applicable) == 0 {
		return errors.New("no scanners configured for artifact type")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	var mu sync.Mutex
	results := make([]ScannerResult, 0, len(applicable))
	wg := sync.WaitGroup{}
	ch := make(chan struct{}, o.workerPoolSize)

	for _, s := range applicable {
		wg.Add(1)
		ch <- struct{}{}
		sc := s
		go func() {
			defer wg.Done()
			defer func() { <-ch }()
			res, err := sc.Scan(ctx, job.ArtifactPath, job.ArtifactType)
			if err != nil {
				// log and continue
				mu.Lock()
				results = append(results, ScannerResult{Tool: sc.Name(), OutputRaw: json.RawMessage([]byte(fmtError(err))), Summary: map[string]interface{}{"error": err.Error()}})
				mu.Unlock()
				return
			}
			mu.Lock()
			// Convert ScanResult to ScannerResult
			scannerResult := ScannerResult{
				Tool:      sc.Name(),
				OutputRaw: json.RawMessage([]byte(fmtError(nil))), // placeholder
				Summary: map[string]interface{}{
					"scanner_name":    res.ScannerName,
					"scanner_version": res.ScannerVersion,
					"artifact_type":   res.ArtifactType,
					"vulnerabilities": res.Vulnerabilities,
					"summary":         res.Summary,
					"scan_duration":   res.ScanDuration,
				},
			}
			results = append(results, scannerResult)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// persist aggregated results
	if err := o.outStore.SaveScanResults(ctx, job.JobID, results); err != nil {
		return err
	}

	// Build policy input
	policyInput := map[string]interface{}{
		"artifact_id":   job.ArtifactID,
		"tenant_id":     job.TenantID,
		"artifact_type": job.ArtifactType,
		"scanners":      results,
	}
	// evaluate policy
	decision, err := o.policy.Evaluate(ctx, policyInput)
	if err != nil {
		// mark failed policy evaluation
		o.outStore.MarkJobCompleted(ctx, job.JobID, "policy_error")
		return err
	}

	// take action based on decision
	switch decision.Action {
	case "allow", "warn":
		o.outStore.MarkJobCompleted(ctx, job.JobID, "completed")
	case "quarantine":
		o.outStore.MarkJobCompleted(ctx, job.JobID, "quarantined")
	case "block":
		o.outStore.MarkJobCompleted(ctx, job.JobID, "blocked")
	default:
		o.outStore.MarkJobCompleted(ctx, job.JobID, "completed")
	}
	return nil
}

func fmtError(err error) string {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	return string(b)
}
