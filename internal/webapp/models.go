package webapp

import "time"

type Config struct {
	OpenClawBaseURL  string `json:"openclaw_base_url"`
	OpenClawToken    string `json:"openclaw_token,omitempty"`
	DiscordURL       string `json:"discord_url,omitempty"`
	DiscordWidgetURL string `json:"discord_widget_url,omitempty"`
	DiscordBotID     string `json:"discord_bot_id,omitempty"`
	GitHubRepo       string `json:"github_repo"`
	ClawHubBaseURL   string `json:"clawhub_base_url"`
	LLMAPIKey        string `json:"llm_api_key,omitempty"`
	LLMLocked        bool   `json:"llm_locked"`
}

type Project struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Plan         string          `json:"plan"`
	Docs         string          `json:"docs"`
	Branches     []ProjectBranch `json:"branches"`
	Requirements []Requirement   `json:"requirements"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type ProjectBranch struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Requirement struct {
	ID          string    `json:"id"`
	BranchID    string    `json:"branch_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	AgentID     string    `json:"agent_id,omitempty"`
	Prompt      string    `json:"prompt,omitempty"`
	CommitID    string    `json:"commit_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OpenClawStatus struct {
	Online    bool   `json:"online"`
	BaseURL   string `json:"base_url"`
	AgentID   string `json:"agent_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
	CheckedAt string `json:"checked_at"`
}

type TailscaleDevice struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DNSName     string `json:"dns_name,omitempty"`
	OS          string `json:"os,omitempty"`
	IP          string `json:"ip,omitempty"`
	Online      bool   `json:"online"`
	Active      bool   `json:"active"`
	Self        bool   `json:"self"`
	LastSeen    string `json:"last_seen,omitempty"`
	OpenClawURL string `json:"openclaw_url,omitempty"`
}

type TailscaleDeviceStatus struct {
	Tailnet   string            `json:"tailnet,omitempty"`
	Devices   []TailscaleDevice `json:"devices"`
	Error     string            `json:"error,omitempty"`
	CheckedAt string            `json:"checked_at"`
}

type SkillSummary struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	Version      string   `json:"version,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Runtime      string   `json:"runtime,omitempty"`
	Entrypoint   string   `json:"entrypoint,omitempty"`
	Description  string   `json:"description,omitempty"`
}

type SkillInstallRequest struct {
	SkillID string `json:"skill_id"`
	AgentID string `json:"agent_id,omitempty"`
}

type SkillInstallResult struct {
	SkillID string     `json:"skill_id"`
	Result  SendResult `json:"result"`
}

type PluginRuntime struct {
	Name        string   `json:"name"`
	Extensions  []string `json:"extensions"`
	CommandHint string   `json:"command_hint"`
}

type PromptResult struct {
	ProjectID     string   `json:"project_id"`
	RequirementID string   `json:"requirement_id"`
	Segments      []string `json:"segments"`
	ArtifactPath  string   `json:"artifact_path"`
	DiscordText   string   `json:"discord_text"`
	Prompt        string   `json:"prompt,omitempty"`
}

type ProjectPromptRequest struct {
	RequirementID string                        `json:"requirement_id"`
	Blocks        []map[string]any              `json:"blocks"`
	TencentDocs   []RequirementEditorTencentDoc `json:"tencentDocs"`
	Attachments   []RequirementEditorAttachment `json:"attachments"`
	PlainText     string                        `json:"plainText"`
	LLM           LLMRequest                    `json:"llm"`
}

type LLMRequest struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url,omitempty"`
}

type LLMTestRequest struct {
	LLM LLMRequest `json:"llm"`
}

type RequirementEditorPromptRequest struct {
	Blocks      []map[string]any              `json:"blocks"`
	TencentDocs []RequirementEditorTencentDoc `json:"tencentDocs"`
	Attachments []RequirementEditorAttachment `json:"attachments"`
	PlainText   string                        `json:"plainText"`
}

type RequirementEditorTencentDoc struct {
	Type            string `json:"type"`
	Title           string `json:"title"`
	URL             string `json:"url"`
	ReadInstruction string `json:"readInstruction"`
}

type RequirementEditorAttachment struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MIMEType   string `json:"mimeType"`
	Size       int64  `json:"size"`
	Kind       string `json:"kind"`
	URL        string `json:"url,omitempty"`
	StorageKey string `json:"storageKey,omitempty"`
	Source     string `json:"source"`
}

type SendResult struct {
	Sent       bool   `json:"sent"`
	Endpoint   string `json:"endpoint"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

type BoardUpdate struct {
	RequirementIDs []string `json:"requirement_ids"`
	BranchID       string   `json:"branch_id,omitempty"`
	AgentID        string   `json:"agent_id,omitempty"`
	CommitID       string   `json:"commit_id"`
	Status         string   `json:"status"`
}
