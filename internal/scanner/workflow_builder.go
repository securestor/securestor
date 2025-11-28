package scanner

import (
	"fmt"

	"github.com/securestor/securestor/internal/config"
)

// BuildWorkflowRegistry creates and configures a workflow registry from config
func BuildWorkflowRegistry(workflowConfig *config.WorkflowConfig) (*WorkflowRegistry, error) {
	registry := NewWorkflowRegistry()

	for _, workflowDef := range workflowConfig.Workflows {
		if !workflowDef.Enabled {
			continue
		}

		// Build scanners list
		scanners := make([]Scanner, 0, len(workflowDef.Scanners))
		for _, scannerName := range workflowDef.Scanners {
			var s Scanner
			switch scannerName {
			case "syft":
				s = NewSyftAdapter("syft")
			case "grype":
				s = NewGrypeAdapter("grype")
			case "trivy":
				s = NewTrivyAdapter("trivy")
			case "depscan":
				s = NewDepScanAdapter("depscan")
			case "osv":
				s = NewOSVScanner()
			case "trufflehog":
				s = NewTruffleHogScanner()
			case "bandit":
				s = NewBanditScanner()
			default:
				return nil, fmt.Errorf("unknown scanner: %s", scannerName)
			}
			scanners = append(scanners, s)
		}

		// Register workflow
		workflow := &Workflow{
			Name:          workflowDef.Name,
			ArtifactTypes: workflowDef.ArtifactTypes,
			Scanners:      scanners,
			PolicyPath:    workflowDef.PolicyPath,
		}

		registry.Register(workflow)
	}

	return registry, nil
}
