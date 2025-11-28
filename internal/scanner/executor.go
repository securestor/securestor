package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ParallelExecutor executes scanners in parallel with proper error handling
type ParallelExecutor struct {
	logger Logger
}

// ExecutionResult represents the result of executing a single scanner
type ExecutionResult struct {
	Scanner   Scanner
	Result    *ScanResult
	Error     error
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(logger Logger) *ParallelExecutor {
	return &ParallelExecutor{
		logger: logger,
	}
}

// Execute runs scanners in parallel with proper synchronization
func (e *ParallelExecutor) Execute(ctx context.Context, request *ScanRequest, scanners []Scanner) ([]*ScanResult, error) {
	if len(scanners) == 0 {
		return nil, fmt.Errorf("no scanners provided")
	}

	e.logger.Printf("[EXECUTOR] Starting parallel execution of %d scanners", len(scanners))

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	// Create channels for results and errors
	resultChan := make(chan *ExecutionResult, len(scanners))

	// Start scanners in parallel
	var wg sync.WaitGroup
	for i, scanner := range scanners {
		wg.Add(1)
		go func(idx int, s Scanner) {
			defer wg.Done()
			e.executeSingleScanner(execCtx, idx, s, request, resultChan)
		}(i, scanner)
	}

	// Wait for all scanners to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []*ScanResult
	var errors []error
	var executionResults []*ExecutionResult

	for execResult := range resultChan {
		executionResults = append(executionResults, execResult)

		if execResult.Error != nil {
			e.logger.Printf("[EXECUTOR] Scanner %s failed: %v", execResult.Scanner.Name(), execResult.Error)
			errors = append(errors, execResult.Error)

			// If FailFast is enabled, cancel remaining scanners
			if request.Options.FailFast {
				cancel()
				break
			}
		} else {
			e.logger.Printf("[EXECUTOR] Scanner %s completed successfully: %d vulnerabilities in %v",
				execResult.Scanner.Name(), len(execResult.Result.Vulnerabilities), execResult.Duration)
			results = append(results, execResult.Result)
		}
	}

	// Log execution summary
	e.logExecutionSummary(executionResults)

	// Check if we have any results
	if len(results) == 0 && len(errors) > 0 {
		if !request.Options.SkipUnavailable {
			return nil, fmt.Errorf("all scanners failed: %v", errors)
		}
	}

	return results, nil
}

// executeSingleScanner executes a single scanner with proper error handling
func (e *ParallelExecutor) executeSingleScanner(ctx context.Context, index int, scanner Scanner, request *ScanRequest, resultChan chan<- *ExecutionResult) {
	startTime := time.Now()

	e.logger.Printf("[EXECUTOR] Starting scanner %d: %s for %s", index, scanner.Name(), request.ArtifactPath)

	execResult := &ExecutionResult{
		Scanner:   scanner,
		StartTime: startTime,
	}

	// Check if scanner is available
	if !scanner.IsAvailable() {
		execResult.Error = fmt.Errorf("scanner %s is not available", scanner.Name())
		execResult.EndTime = time.Now()
		execResult.Duration = execResult.EndTime.Sub(execResult.StartTime)
		resultChan <- execResult
		return
	}

	// Execute scanner with retry logic
	maxRetries := request.Options.RetryCount
	if maxRetries == 0 {
		maxRetries = 1 // At least one attempt
	}

	var lastError error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			e.logger.Printf("[EXECUTOR] Retrying scanner %s (attempt %d/%d)", scanner.Name(), attempt, maxRetries)
		}

		// Create a timeout context for this specific scanner
		scanCtx, scanCancel := context.WithTimeout(ctx, 2*time.Minute)

		result, err := scanner.Scan(scanCtx, request.ArtifactPath, request.ArtifactType)
		scanCancel()

		if err == nil {
			execResult.Result = result
			execResult.EndTime = time.Now()
			execResult.Duration = execResult.EndTime.Sub(execResult.StartTime)
			resultChan <- execResult
			return
		}

		lastError = err

		// Check if context was cancelled (no point in retrying)
		if ctx.Err() != nil {
			break
		}

		// Wait before retry (exponential backoff)
		if attempt < maxRetries {
			retryDelay := time.Duration(attempt) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				lastError = ctx.Err()
				break
			case <-time.After(retryDelay):
				continue
			}
		}
	}

	execResult.Error = fmt.Errorf("scanner %s failed after %d attempts: %w", scanner.Name(), maxRetries, lastError)
	execResult.EndTime = time.Now()
	execResult.Duration = execResult.EndTime.Sub(execResult.StartTime)
	resultChan <- execResult
}

// logExecutionSummary logs a summary of the execution results
func (e *ParallelExecutor) logExecutionSummary(results []*ExecutionResult) {
	successful := 0
	failed := 0
	totalVulns := 0
	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.Error != nil {
			failed++
		} else {
			successful++
			if result.Result != nil {
				totalVulns += len(result.Result.Vulnerabilities)
			}
		}
		totalDuration += result.Duration
	}

	e.logger.Printf("[EXECUTOR] Execution Summary: %d successful, %d failed, %d total vulnerabilities, %v total time",
		successful, failed, totalVulns, totalDuration)

	// Log individual scanner performance
	for _, result := range results {
		status := "SUCCESS"
		details := "0 vulnerabilities" // default

		if result.Error != nil {
			status = "FAILED"
			details = result.Error.Error()
		} else if result.Result != nil {
			details = fmt.Sprintf("%d vulnerabilities", len(result.Result.Vulnerabilities))
		}

		e.logger.Printf("[EXECUTOR] Scanner %s: %s in %v - %s",
			result.Scanner.Name(), status, result.Duration, details)
	}
}
