// workflow.go
package scanner

type Workflow struct {
	Name          string
	ArtifactTypes []string
	Scanners      []Scanner
	PolicyPath    string
	PreSteps      []func(ScanJob) error
	PostSteps     []func(ScanJob, []ScannerResult) error
}

type WorkflowRegistry struct {
	workflows map[string]*Workflow
}

func NewWorkflowRegistry() *WorkflowRegistry {
	return &WorkflowRegistry{workflows: make(map[string]*Workflow)}
}

func (r *WorkflowRegistry) Register(wf *Workflow) {
	for _, t := range wf.ArtifactTypes {
		r.workflows[t] = wf
	}
}

func (r *WorkflowRegistry) Get(artifactType string) *Workflow {
	return r.workflows[artifactType]
}

// Scanner adapter functions for workflow registration

func NewSyftAdapter(name string) Scanner {
	return NewSyftScanner()
}

func NewGrypeAdapter(name string) Scanner {
	return NewSyftScanner() // Syft scanner includes Grype functionality
}

func NewTrivyAdapter(name string) Scanner {
	return NewTrivyScanner()
}

func NewDepScanAdapter(name string) Scanner {
	return NewDepScanScanner()
}
