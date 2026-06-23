package controllayer

type CodeClass string

const (
	ClassSafeModule   CodeClass = "safe_module_code"
	ClassInfra        CodeClass = "infrastructure_code"
	ClassSystemDesign CodeClass = "system_design_code"
)

type ValidationStatus string

const (
	ValidationAllowed ValidationStatus = "allowed"
	ValidationBlocked ValidationStatus = "blocked"
)

type PipelineInput struct {
	Prompt        string `json:"prompt"`
	ApproveSystem bool   `json:"approve_system"`
}

type Generation struct {
	Content string `json:"content"`
}

type Validation struct {
	Class   CodeClass        `json:"class"`
	Status  ValidationStatus `json:"status"`
	Reasons []string         `json:"reasons"`
}

type Artifact struct {
	ID         string     `json:"id"`
	Prompt     string     `json:"prompt"`
	Generation Generation `json:"generation"`
	Validation Validation `json:"validation"`
}
