package webapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type Server struct {
	store     Store
	connector Connector
	staticDir string
}

func NewServer(store Store, staticDir string) Server {
	return Server{
		store:     store,
		connector: NewConnector(),
		staticDir: staticDir,
	}
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(s.staticDir)))
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/openclaw/status", s.handleOpenClawStatus)
	mux.HandleFunc("/api/tailscale/devices", s.handleTailscaleDevices)
	mux.HandleFunc("/api/clawhub/skills", s.handleClawHubSkills)
	mux.HandleFunc("/api/clawhub/skills/", s.handleClawHubSkillActions)
	mux.HandleFunc("/api/plugin-runtimes", s.handlePluginRuntimes)
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/projects/", s.handleProjectActions)
	mux.HandleFunc("/api/github/push", s.handleGitHubPush)
	return mux
}

func (s Server) handleClawHubSkillActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/clawhub/skills/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "skill not found")
		return
	}
	skillID := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		s.handleClawHubSkillDetail(w, r, skillID)
		return
	}
	if len(parts) == 2 && parts[1] == "install" && r.Method == http.MethodPost {
		s.handleClawHubSkillInstall(w, r, skillID)
		return
	}
	writeError(w, http.StatusNotFound, "unknown skill action")
}

func (s Server) handleClawHubSkillDetail(w http.ResponseWriter, r *http.Request, skillID string) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	skill, err := s.connector.GetClawHubSkill(r.Context(), state.Config.ClawHubBaseURL, skillID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, skill)
}

func (s Server) handleClawHubSkillInstall(w http.ResponseWriter, r *http.Request, skillID string) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var input SkillInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	input.SkillID = skillID
	result, err := s.connector.InstallSkillOnOpenClaw(r.Context(), state.Config.OpenClawBaseURL, state.Config.OpenClawToken, state.Config.ClawHubBaseURL, input)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, result)
}

func (s Server) handlePluginRuntimes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, SupportedPluginRuntimes())
}

func (s Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, state)
}

func (s Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, state.Config)
	case http.MethodPut:
		var cfg Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		state.Config = cfg
		if err := s.store.Save(state); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, cfg)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s Server) handleOpenClawStatus(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, s.connector.CheckOpenClaw(r.Context(), state.Config.OpenClawBaseURL))
}

func (s Server) handleTailscaleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	writeJSON(w, ReadTailscaleDevices(ctx))
}

func (s Server) handleClawHubSkills(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	skills, err := s.connector.ListClawHubSkills(r.Context(), state.Config.ClawHubBaseURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, skills)
}

func (s Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, state.Projects)
	case http.MethodPost:
		var input Project
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.TrimSpace(input.Name) == "" {
			writeError(w, http.StatusBadRequest, "project name is required")
			return
		}
		project := UpsertProject(&state, input)
		if err := s.store.Save(state); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, project)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s Server) handleProjectActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "unknown project action")
		return
	}
	projectID := parts[0]
	action := parts[1]
	switch action {
	case "branches":
		s.handleBranches(w, r, projectID)
	case "requirements":
		s.handleRequirements(w, r, projectID)
	case "prompt":
		s.handlePrompt(w, r, projectID)
	case "send":
		s.handleSendPrompt(w, r, projectID)
	case "board":
		s.handleBoardUpdate(w, r, projectID)
	default:
		writeError(w, http.StatusNotFound, "unknown project action")
	}
}

func (s Server) handleBranches(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	branch, ok := AddBranch(&state, projectID, input.Name)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := s.store.Save(state); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, branch)
}

func (s Server) handleRequirements(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var input Requirement
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		writeError(w, http.StatusBadRequest, "requirement title is required")
		return
	}
	req, ok := AddRequirement(&state, projectID, input)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := s.store.Save(state); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, req)
}

func (s Server) handlePrompt(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input struct {
		RequirementID string `json:"requirement_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	project, ok := FindProject(state, projectID)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	req, ok := FindRequirement(project, input.RequirementID)
	if !ok {
		writeError(w, http.StatusNotFound, "requirement not found")
		return
	}
	prompt := BuildPrompt(project, req)
	result, err := StorePromptArtifact(project, req, prompt, "artifacts")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result.DiscordText = BuildDiscordMessageForBot(project, req, prompt, state.Config.DiscordBotID)
	writeJSON(w, result)
}

func (s Server) handleSendPrompt(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input struct {
		RequirementID string `json:"requirement_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	project, ok := FindProject(state, projectID)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	req, ok := FindRequirement(project, input.RequirementID)
	if !ok {
		writeError(w, http.StatusNotFound, "requirement not found")
		return
	}
	prompt := BuildPrompt(project, req)
	result, err := s.connector.SendOpenClawPrompt(r.Context(), state.Config.OpenClawBaseURL, state.Config.OpenClawToken, prompt)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, result)
}

func (s Server) handleBoardUpdate(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input BoardUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(input.RequirementIDs) == 0 {
		writeError(w, http.StatusBadRequest, "requirement_ids is required")
		return
	}
	state, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, ok := UpdateRequirements(&state, projectID, input)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := s.store.Save(state); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, updated)
}

func (s Server) handleGitHubPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := runGit("status", "--short"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := runGit("push"); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, map[string]string{"status": "pushed"})
}

func runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	if args[0] == "status" && strings.TrimSpace(string(out)) != "" {
		return errors.New("working tree has uncommitted changes; commit before pushing")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
