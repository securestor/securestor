package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OPA client with configurable policy path
type OPAClient struct {
	URL        string // base URL, e.g. http://opa:8181
	PolicyPath string // e.g. /v1/data/securestor/policy/allow
	http       *http.Client
}

func NewOPAClient(baseURL, policyPath string) *OPAClient {
	return &OPAClient{URL: baseURL, PolicyPath: policyPath, http: &http.Client{Timeout: 10 * time.Second}}
}

func (o *OPAClient) Evaluate(ctx context.Context, artifactMeta map[string]interface{}) (Decision, error) {
	input := map[string]interface{}{"input": artifactMeta}
	b, _ := json.Marshal(input)
	req, err := http.NewRequestWithContext(ctx, "POST", o.URL+o.PolicyPath, bytes.NewReader(b))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return Decision{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Decision{}, fmt.Errorf("opa error status: %d", resp.StatusCode)
	}
	var out struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Decision{}, err
	}
	// map result to Decision
	dec := Decision{Allow: true, Action: "allow", Reason: ""}
	if a, ok := out.Result["allow"].(bool); ok {
		dec.Allow = a
	}
	if act, ok := out.Result["action"].(string); ok {
		dec.Action = act
	}
	if r, ok := out.Result["reason"].(string); ok {
		dec.Reason = r
	}
	return dec, nil
}
