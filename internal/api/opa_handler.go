package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/securestor/securestor/internal/scanner"
)

// OPATestRequest represents the input for OPA policy testing
type OPATestRequest struct {
	Input map[string]interface{} `json:"input"`
}

// OPATestResponse represents the output from OPA policy evaluation
type OPATestResponse struct {
	Decision scanner.Decision `json:"decision"`
	Status   string           `json:"status"`
	Message  string           `json:"message,omitempty"`
}

// handleOPATest provides a simple endpoint to test OPA integration
func (s *Server) handleOPATest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OPATestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if OPA is enabled
	opaEnabled := os.Getenv("OPA_ENABLED") == "true"
	if !opaEnabled {
		response := OPATestResponse{
			Status:  "disabled",
			Message: "OPA integration is disabled",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get OPA URL from environment
	opaURL := os.Getenv("OPA_URL")
	if opaURL == "" {
		opaURL = "http://opa:8181" // default
	}

	// Create OPA client
	opaClient := scanner.NewOPAClient(opaURL, "/v1/data/securestor/policy/result")

	// Evaluate policy
	decision, err := opaClient.Evaluate(r.Context(), req.Input)
	if err != nil {
		s.logger.Printf("OPA evaluation error: %v", err)
		response := OPATestResponse{
			Status:  "error",
			Message: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := OPATestResponse{
		Decision: decision,
		Status:   "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOPAHealth provides a health check for the OPA service
func (s *Server) handleOPAHealth(w http.ResponseWriter, r *http.Request) {
	opaURL := os.Getenv("OPA_URL")
	if opaURL == "" {
		opaURL = "http://opa:8181"
	}

	// Simple health check
	client := &http.Client{}
	resp, err := client.Get(opaURL + "/health")
	if err != nil {
		response := map[string]interface{}{
			"status":  "unhealthy",
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		response := map[string]interface{}{
			"status": "healthy",
			"url":    opaURL,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		response := map[string]interface{}{
			"status":     "unhealthy",
			"statusCode": resp.StatusCode,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
	}
}
