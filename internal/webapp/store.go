package webapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type State struct {
	Config   Config    `json:"config"`
	Projects []Project `json:"projects"`
}

type Store struct {
	path string
}

func NewStore(path string) Store {
	return Store{path: path}
}

func DefaultStatePath() string {
	if path := os.Getenv("MUTESOLO_WEB_STATE"); path != "" {
		return path
	}
	return ".openclaw/web-state.json"
}

func (s Store) Load() (State, error) {
	data, err := os.ReadFile(s.path)
	if err == nil {
		var state State
		if err := json.Unmarshal(data, &state); err != nil {
			return State{}, fmt.Errorf("decode web state: %w", err)
		}
		ensureStateDefaults(&state)
		return state, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return State{}, fmt.Errorf("read web state: %w", err)
	}
	state := State{
		Config: Config{
			OpenClawBaseURL: "http://100.x.y.z:18800",
			ClawHubBaseURL:  "https://clawhub.example.com",
		},
		Projects: []Project{},
	}
	return state, s.Save(state)
}

func (s Store) Save(state State) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create web state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode web state: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write web state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace web state: %w", err)
	}
	return nil
}

func UpsertProject(state *State, input Project) Project {
	now := time.Now().UTC()
	input.Name = strings.TrimSpace(input.Name)
	if input.ID == "" {
		input.ID = newID(input.Name)
		input.CreatedAt = now
	}
	input.UpdatedAt = now
	if len(input.Branches) == 0 {
		input.Branches = []ProjectBranch{{ID: "main", Name: "Main", CreatedAt: now}}
	}
	if input.Requirements == nil {
		input.Requirements = []Requirement{}
	}
	for i, project := range state.Projects {
		if project.ID == input.ID {
			if input.CreatedAt.IsZero() {
				input.CreatedAt = project.CreatedAt
			}
			if input.Requirements == nil {
				input.Requirements = project.Requirements
			}
			if len(input.Branches) == 0 {
				input.Branches = project.Branches
			}
			state.Projects[i] = input
			ensureProjectDefaults(&state.Projects[i])
			return state.Projects[i]
		}
	}
	state.Projects = append(state.Projects, input)
	ensureProjectDefaults(&state.Projects[len(state.Projects)-1])
	sortProjects(state.Projects)
	for _, project := range state.Projects {
		if project.ID == input.ID {
			return project
		}
	}
	return input
}

func AddRequirement(state *State, projectID string, input Requirement) (Requirement, bool) {
	now := time.Now().UTC()
	for pi := range state.Projects {
		if state.Projects[pi].ID != projectID {
			continue
		}
		ensureProjectDefaults(&state.Projects[pi])
		input.Title = strings.TrimSpace(input.Title)
		if input.ID == "" {
			input.ID = newID(input.Title)
			input.CreatedAt = now
		}
		if input.BranchID == "" {
			input.BranchID = state.Projects[pi].Branches[0].ID
		}
		if input.Priority == "" {
			input.Priority = "low"
		}
		if input.AgentID == "" {
			input.AgentID = "openclaw-a"
		}
		if input.Status == "" {
			input.Status = "draft"
		}
		input.UpdatedAt = now
		state.Projects[pi].Requirements = append(state.Projects[pi].Requirements, input)
		state.Projects[pi].UpdatedAt = now
		return input, true
	}
	return Requirement{}, false
}

func UpdateRequirementDetails(state *State, projectID string, reqID string, input Requirement) (Requirement, bool) {
	now := time.Now().UTC()
	for pi := range state.Projects {
		if state.Projects[pi].ID != projectID {
			continue
		}
		ensureProjectDefaults(&state.Projects[pi])
		for ri := range state.Projects[pi].Requirements {
			req := &state.Projects[pi].Requirements[ri]
			if req.ID != reqID {
				continue
			}
			if strings.TrimSpace(input.Title) != "" {
				req.Title = strings.TrimSpace(input.Title)
			}
			req.Description = strings.TrimSpace(input.Description)
			if strings.TrimSpace(input.Priority) != "" {
				req.Priority = strings.TrimSpace(input.Priority)
			}
			if strings.TrimSpace(input.AgentID) != "" {
				req.AgentID = strings.TrimSpace(input.AgentID)
			}
			req.UpdatedAt = now
			state.Projects[pi].UpdatedAt = now
			return *req, true
		}
		return Requirement{}, false
	}
	return Requirement{}, false
}

func AddBranch(state *State, projectID string, name string) (ProjectBranch, bool) {
	now := time.Now().UTC()
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Branch"
	}
	for pi := range state.Projects {
		if state.Projects[pi].ID != projectID {
			continue
		}
		ensureProjectDefaults(&state.Projects[pi])
		branch := ProjectBranch{ID: newID(name), Name: name, CreatedAt: now}
		state.Projects[pi].Branches = append(state.Projects[pi].Branches, branch)
		state.Projects[pi].UpdatedAt = now
		return branch, true
	}
	return ProjectBranch{}, false
}

func UpdateRequirements(state *State, projectID string, update BoardUpdate) ([]Requirement, bool) {
	now := time.Now().UTC()
	ids := make(map[string]bool, len(update.RequirementIDs))
	for _, id := range update.RequirementIDs {
		ids[id] = true
	}
	status := strings.TrimSpace(update.Status)
	updated := make([]Requirement, 0, len(ids))
	for pi := range state.Projects {
		if state.Projects[pi].ID != projectID {
			continue
		}
		ensureProjectDefaults(&state.Projects[pi])
		for ri := range state.Projects[pi].Requirements {
			req := &state.Projects[pi].Requirements[ri]
			if ids[req.ID] {
				if status != "" {
					req.Status = status
				}
				if strings.TrimSpace(update.BranchID) != "" {
					req.BranchID = strings.TrimSpace(update.BranchID)
				}
				if strings.TrimSpace(update.AgentID) != "" {
					req.AgentID = strings.TrimSpace(update.AgentID)
				}
				if strings.TrimSpace(update.CommitID) != "" {
					req.CommitID = strings.TrimSpace(update.CommitID)
				}
				req.UpdatedAt = now
				updated = append(updated, *req)
			}
		}
		state.Projects[pi].UpdatedAt = now
		return updated, true
	}
	return nil, false
}

func ensureStateDefaults(state *State) {
	for i := range state.Projects {
		ensureProjectDefaults(&state.Projects[i])
	}
}

func ensureProjectDefaults(project *Project) {
	if len(project.Branches) == 0 {
		created := project.CreatedAt
		if created.IsZero() {
			created = time.Now().UTC()
		}
		project.Branches = []ProjectBranch{{ID: "main", Name: "Main", CreatedAt: created}}
	}
	for i := range project.Requirements {
		if project.Requirements[i].BranchID == "" {
			project.Requirements[i].BranchID = project.Branches[0].ID
		}
		if project.Requirements[i].Priority == "" {
			project.Requirements[i].Priority = "low"
		}
		if project.Requirements[i].AgentID == "" {
			project.Requirements[i].AgentID = "openclaw-a"
		}
	}
}

func FindProject(state State, id string) (Project, bool) {
	for _, project := range state.Projects {
		if project.ID == id {
			return project, true
		}
	}
	return Project{}, false
}

func FindRequirement(project Project, id string) (Requirement, bool) {
	for _, req := range project.Requirements {
		if req.ID == id {
			return req, true
		}
	}
	return Requirement{}, false
}

func sortProjects(projects []Project) {
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].UpdatedAt.After(projects[j].UpdatedAt)
	})
}

func newID(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	id := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, name)
	id = strings.Trim(id, "-")
	id = strings.Join(strings.FieldsFunc(id, func(r rune) bool { return r == '-' }), "-")
	if id == "" {
		id = "item"
	}
	return fmt.Sprintf("%s-%d", id, time.Now().UTC().UnixNano())
}
