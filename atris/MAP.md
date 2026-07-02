# Mutesolo Code Map

> Generated: 2026-07-01 · Language: Go 1.22 + TypeScript/JS + HTML/CSS · Module: `Mutesolo`
> Project root: `/Users/soda/Documents/Mutesolo`

---

## Quick Reference (Top 25 Symbols)

| # | Symbol | File | Kind | Role |
|---|--------|------|------|------|
| 1 | `cmd/mutesolo-web/main.go` `main()` | `cmd/mutesolo-web/main.go:12` | Entry point | HTTP server bootstrap, listens `:8787` |
| 2 | `cmd/opclawctl/main.go` `main()` | `cmd/opclawctl/main.go:16` | Entry point | CLI tool for coordination ops |
| 3 | `Server` | `internal/webapp/server.go:16` | Struct | HTTP handler hub, 20+ route handlers |
| 4 | `Store` | `internal/webapp/store.go:19` | Struct | JSON-file persistence (state save/load) |
| 5 | `State` | `internal/webapp/store.go:14` | Struct | In-memory webapp state with lock |
| 6 | `Config` | `internal/webapp/models.go:5` | Struct | Server config (OpenClaw, LLM, Discord, GitHub) |
| 7 | `Project` | `internal/webapp/models.go:17` | Struct | Project with branches & requirements |
| 8 | `Requirement` | `internal/webapp/models.go:35` | Struct | Single issue/card with prompt & agent |
| 9 | `Connector` | `internal/webapp/connectors.go:14` | Struct | Outbound HTTP to OpenClaw & ClawHub |
| 10 | `GenerateOpenCodePrompt` | `internal/webapp/llm.go:66` | Func | LLM proxy for OpenCode API |
| 11 | `BuildPrompt` | `internal/webapp/prompt.go:11` | Func | Assembles requirement → prompt text |
| 12 | `SegmentPrompt` | `internal/webapp/prompt.go:34` | Func | Splits long prompts into Discord-safe chunks |
| 13 | `BuildDiscordMessage` | `internal/webapp/prompt.go:71` | Func | Formats prompt for Discord dispatch |
| 14 | `AssetStorage` | `internal/webapp/assets.go:25` | Struct | S3 + local-fallback file upload |
| 15 | `ParseDocument` | `internal/webapp/documents.go:61` | Func | MinerU document parsing pipeline |
| 16 | `ReadTailscaleDevices` | `internal/webapp/tailscale.go:33` | Func | Tailscale `status` JSON reader |
| 17 | `CoordinationState` | `internal/coordination/models.go:61` | Struct | Agent/skill/task/session registry |
| 18 | `CreateTask` | `internal/coordination/core.go:23` | Func | Capability-based task creation |
| 19 | `MatchTask` | `internal/coordination/core.go:48` | Func | Best-agent matching by coverage |
| 20 | `AssignTask` | `internal/coordination/core.go:69` | Func | Assign task → agent, create session |
| 21 | `RunPipeline` | `control_layer/pipeline.go:8` | Func | Code gen pipeline: generate→validate→store |
| 22 | `Classify` | `control_layer/classifier.go:5` | Func | Content classification (safe/infra/design) |
| 23 | `Validate` | `control_layer/validator.go:5` | Func | Prompt+generation validation |
| 24 | `StoreArtifact` | `control_layer/artifacts.go:12` | Func | Persist pipeline artifact to disk |
| 25 | `RequirementEditor` | `webapps/requirement-editor/src/RequirementEditor.tsx:140` | Component | Rich text editor (BlockNote) |

---

## By-Feature Map

### 1. Web Console Server (`internal/webapp/`)

**Core types** — `models.go`
| Symbol | Kind | Role |
|--------|------|------|
| `Config` | struct | Server configuration (OpenClaw URL/token, Discord, GitHub, LLM, ClawHub) |
| `Project` | struct | Named project with plan/docs/branches/requirements |
| `ProjectBranch` | struct | Named branch within a project |
| `Requirement` | struct | Single work item: title, description, priority, status, agent, prompt, commit |
| `OpenClawStatus` | struct | OpenClaw health check result |
| `TailscaleDevice` | struct | Tailscale node info |
| `SkillSummary` | struct | ClawHub skill metadata |
| `PluginRuntime` | struct | Supported plugin runtime descriptor |
| `PromptResult` | struct | Built prompt + segments + artifact metadata |
| `LLMRequest` | struct | LLM provider/model/auth input |
| `BoardUpdate` | struct | Batch requirement status/branch/agent update |
| `RequirementEditorTencentDoc` | struct | Tencent Doc reference for editor |
| `RequirementEditorAttachment` | struct | Asset attachment for editor |

**HTTP Routing** — `server.go`
```
GET /                             → static files (web/)
GET /assets/                      → asset fallback file server
GET /apps/requirement-editor/     → SPA static build
GET /api/state                    → full state dump
POST /api/config                  → update config
POST /api/openclaw/status         → check OpenClaw health
POST /api/tailscale/devices       → fetch Tailscale device list
POST /api/clawhub/skills          → list ClawHub skills
GET  /api/clawhub/skills/{id}     → skill detail
POST /api/clawhub/skills/{id}/install → install skill
POST /api/plugin-runtimes         → list supported runtimes
POST /api/assets                  → upload asset (S3 or local)
POST /api/documents/parse         → parse document via MinerU
POST /api/llm/test                → test LLM connectivity
POST /api/projects                → list / create projects
GET  /api/projects/{id}           → project detail
POST /api/projects/{id}/branches  → list / add branches
GET  /api/projects/{id}/requirements       → list/add requirements
GET  /api/projects/{id}/requirements/{rid} → requirement detail
PUT  /api/projects/{id}/requirements/{rid} → update requirement
POST /api/projects/{id}/prompt    → build prompt (LLM-assisted)
POST /api/projects/{id}/send      → send prompt to Discord/OpenClaw
POST /api/projects/{id}/board     → batch board update
POST /api/generate-prompt         → standalone prompt generation from editor context
POST /api/github/push             → git push webhook
```

**Persistence** — `store.go`
| Func | Role |
|------|------|
| `NewStore`, `Load`, `Save`, `WithState` | Thread-safe JSON file persistence (file lock) |
| `UpsertProject` | Create or update project |
| `AddRequirement` | Add requirement to project |
| `UpdateRequirementDetails` | Update single requirement |
| `AddBranch` | Add branch to project |
| `UpdateRequirements` | Batch update: status, branch, agent, commit |
| `FindProject`, `FindRequirement` | Lookup helpers |

**Outbound Connectors** — `connectors.go`
| Func | Role |
|------|------|
| `CheckOpenClaw` | OpenClaw agent card health check |
| `ListClawHubSkills`, `GetClawHubSkill` | ClawHub registry queries |
| `InstallSkillOnOpenClaw` | Skill install via OpenClaw API |
| `SendOpenClawPrompt` | Send prompt text to OpenClaw |

**Prompt Building** — `prompt.go`
| Func | Role |
|------|------|
| `BuildPrompt` | Assemble project+requirement → structured prompt |
| `SegmentPrompt` | Split >1900-char prompts for Discord |
| `StorePromptArtifact` | Write prompt JSON to artifacts dir |
| `BuildDiscordMessage` | Format prompt as Discord message |
| `BuildDiscordMessageForBot` | Same but with bot mention for A2A |
| `BuildRequirementEditorPrompt` | Combine editor blocks + Tencent docs + attachments → prompt text |
| `BuildLLMPromptInput` | Full LLM input: project context + requirement + editor context |
| `isNonLocalURL`, `agentDisplayName`, `fallback` | Helpers |

**LLM Integration** — `llm.go`
| Func | Role |
|------|------|
| `GenerateOpenCodePrompt` | Send prompt to OpenCode API |
| `TestOpenCodeConnection` | Test LLM endpoint connectivity |
| `LLMRequestFromConfig` | Extract LLM config from server config |
| `MergeLLMRequest` | Merge saved config with request overrides |
| `generateOpenCodePrompt` | Internal: try model candidates with fallback chain |
| `openCodeCandidateModels` | Build fallback model list |
| `requestOpenCodePrompt` | Raw HTTP call to OpenCode |
| `isRetryableOpenCodeModelError` | Detect retryable model errors |

**Document Parsing** — `documents.go`
| Func | Role |
|------|------|
| `handleDocumentParse` | HTTP handler (POST /api/documents/parse) |
| `ParseDocument` | Orchestrate document parsing via MinerU |
| `prepareDocumentInput` | Resolve URL or local path → download/copy |
| `downloadDocument` | HTTP download to temp dir |
| `findMinerUOutputs` | Locate MinerU markdown/results in output dir |
| `safeFilename` | Sanitize filenames |

**Asset Storage** — `assets.go`
| Symbol | Role |
|--------|------|
| `AssetStorage` | Struct with S3 config (env-configured) |
| `Upload` | Upload to S3, fallback to local |
| `putObject` | Raw S3 PutObject via presigned |
| `signS3Request` | AWS SigV4 signing |
| `cleanupLocalAssets` | Age-based local cleanup |
| `AssetStorageFromEnv` | Factory from MUTESOLO_* env vars |

**Tailscale** — `tailscale.go`
| Func | Role |
|------|------|
| `ReadTailscaleDevices` | Execute `tailscale status --json` and parse |
| `tailscaleDeviceFromNode` | Map raw node → TailscaleDevice |
| `openClawURLForTailscaleIP` | Build `http://100.x.x.x` URL from IP |

### 2. Coordination Engine (`internal/coordination/`)

**Models** — `models.go`
| Symbol | Role |
|--------|------|
| `AgentStatus` (enum) | online / offline / busy |
| `TaskStatus` (enum) | pending / matched / assigned |
| `SessionStatus` (enum) | active / closed |
| `Agent` | Agent with address, skills, status |
| `Skill` | Skill with capabilities & version |
| `Task` | Task with required capabilities |
| `Session` | Agent-task binding |
| `Event` | State-change event with timestamp |
| `State` | Aggregate: agents, skills, tasks, sessions, events |
| `InitialState()` | Factory with 3 mock agents + 5 mock skills |

**Core Logic** — `core.go`
| Func | Role |
|------|------|
| `CreateTask` | Create task with normalized capabilities |
| `MatchTask` | Find highest-coverage online agent |
| `AssignTask` | Assign task → agent, create session + events |
| `bestAgentForTask` | Internal: score agents by cap coverage |
| `coverage` | Count matching capabilities |
| `normalizeCaps`, `uniqueCaps`, `capSet` | Capability normalization helpers |

**Persistence** — `store.go`
| Func | Role |
|------|------|
| `NewStore`, `Load`, `Save`, `WithState` | Same pattern as webapp store |

### 3. Control Layer Pipeline (`control_layer/`)

**Models** — `models.go`
| Const/Type | Role |
|------------|------|
| `CodeClass`: `safe_module_code`, `infrastructure_code`, `system_design_code` | Content classification enum |
| `ValidationStatus`: `allowed`, `blocked` | Validation result |
| `PipelineInput` | Prompt + system-design flag |
| `Generation`, `Validation`, `Artifact` | Pipeline stage outputs |

**Pipeline** — `pipeline.go`
| Func | Role |
|------|------|
| `RunPipeline` | Generate → Validate → Store artifact |

**Stage 1: Generate** — `generator.go`
| Func | Role |
|------|------|
| `Generate` | Read template, interpolate prompt → generation |
| `Output` | Read generation output from template |

**Stage 2: Classify** — `classifier.go`
| Func | Role |
|------|------|
| `Classify` | Heuristic content classification (keyword signals) |
| `containsAny` | Substring match helper |

**Stage 3: Validate** — `validator.go`
| Func | Role |
|------|------|
| `Validate` | Block system design unless approved; allow safe/infra |

**Artifact** — `artifacts.go`
| Func | Role |
|------|------|
| `StoreArtifact` | Serialize artifact JSON to file |
| `NewArtifactID` | Deterministic ID from prompt+generation+validation |

### 4. CLI Tool (`cmd/opclawctl/`)

| Func | Role |
|------|------|
| `main()` | CLI dispatch |
| `run()` | Root command handler |
| `pipelineCommand()` | `opclawctl pipeline` subcommand |
| `pipelineRunCommand()` | `opclawctl pipeline run` subcommand |
| `agentsCommand()` | List agents from coordination state |
| `skillsCommand()` | List skills from coordination state |
| `tasksCommand()` | List tasks |
| `createTaskCommand()` | Create task via coordination engine |
| `matchTaskCommand()` | Match task to best agent |
| `assignTaskCommand()` | Assign task to agent |
| `eventsCommand()` | List coordination events |
| `splitCaps`, `newTable`, `printUsage` | CLI helpers |

### 5. Frontend (`web/` + `webapps/`)

**Legacy Console** — `web/app.js` (~1224 lines)
| Func | Role |
|------|------|
| `state` (global) | App state: projects, config, tailscale, skills |
| `renderProjects`, `renderSideProjects` | Project list sidebar & main view |
| `renderBranches` | Branch list in board view |
| `renderBoard`, `renderIssueCard` | Kanban-like board |
| `renderDiscordWidget`, `showDiscordPreview`, `openDiscord` | Discord integration UI |
| `renderOpenClawStrip` | OpenClaw health status bar |
| `renderSkills` | Skill management UI |
| `readLLMInputs`, `setLLMEditMode` | LLM config form |
| `renderBoard`, `renderBranchList`, `renderBoardMode` | Board/branch view toggle |
| `openRequirementModal`, `openBranchModal` | CRUD modals |
| `renderAgentSelect`, `renderAgentSelects` | Agent dropdowns |
| `currentProject`, `currentRequirement` | State helpers |
| `renderPromptResult`, `setPromptProgress`, `startPromptProgress` | Prompt progress UI |
| `renderTaskView`, `renderTaskDetail` | Coordination task view |
| `requestRequirementEditorContext` | Context bridge to editor SPA |
| `editorContextRequests` (Map) | Pending editor context request queue |

**Static assets** — `web/index.html`, `web/styles.css`

**Requirement Editor SPA** — `webapps/requirement-editor/`
| Symbol | Role |
|--------|------|
| `RequirementEditor` (default export) | Main React component (BlockNote-based rich text) |
| `draftKeyFromSearch` | Parse draft key from URL search params |
| `starterBlocksFromSearch` | Parse starter blocks from URL search |
| `readDraft`, `sanitizeAttachments` | Draft persistence & sanitization |
| `textFromInlineContent`, `blockToText`, `buildPlainText` | Block → plain text extraction |
| `emptyTencentDoc` | Tencent Doc factory |

---

## By-Concern Map

### Configuration & Environment
| Concern | Files | Key Symbols |
|---------|-------|-------------|
| Server config | `models.go:5` (Config) | `openclaw_base_url`, `llm_api_key`, `discord_url`, `github_repo`, `clawhub_base_url` |
| Env variables | `assets.go:46` | `MUTESOLO_AWS_*`, `MUTESOLO_ASSET_FALLBACK_DIR`, `MUTESOLO_ASSET_CLEANUP_AGE` |
| State path | `store.go:27` | `DefaultStatePath()` → `$HOME/.mutesolo-state.json` |

### HTTP Communication
| Concern | Files | Key Symbols |
|---------|-------|-------------|
| Server | `server.go`, `cmd/mutesolo-web/main.go` | `Handler()` → `http.NewServeMux()`, `/api/*` routes |
| OpenClaw client | `connectors.go` | `CheckOpenClaw`, `SendOpenClawPrompt`, `InstallSkillOnOpenClaw` |
| ClawHub client | `connectors.go` | `ListClawHubSkills`, `GetClawHubSkill` |
| LLM client | `llm.go` | `requestOpenCodePrompt` (HTTP to OpenCode API) |
| Tailscale CLI | `tailscale.go` | Shell exec `tailscale status --json` |
| S3 client | `assets.go` | Raw SigV4-signed HTTP requests |

### Data Flow
```
User (browser) ←→ mutesolo-web (:8787)
    ↓
internal/webapp/
    ├─ server.go (routing + JSON handlers)
    ├─ store.go (JSON file ←→ in-memory state)
    ├─ connectors.go (outbound to OpenClaw/ClawHub)
    ├─ llm.go (OpenCode proxy)
    ├─ prompt.go (prompt assembly)
    ├─ documents.go (MinerU parsing)
    ├─ assets.go (S3/local upload)
    └─ tailscale.go (Tailscan status)
        ↓
internal/coordination/  ← CLI-driven (opclawctl)
    ├─ core.go (task lifecycle)
    └─ store.go (persistence)
        ↓
control_layer/
    └─ pipeline.go (generate → classify → validate → store)
```

### Cross-Cutting Utilities
| Utility | File | Description |
|---------|------|-------------|
| Filename sanitizer | `documents.go:233` | `safeFilename()` — alphanumeric only |
| JSON I/O helpers | `server.go:561-566` | `writeJSON`, `writeError` |
| ID generator | `store.go:287` | `newID()` — slug + timestamp |
| Random hex | `assets.go:251` | `randomHex()` — asset key entropy |
| S3 SigV4 | `assets.go:165` | `signS3Request`, `s3SigningKey`, `hmacSHA256` |

---

## Critical Files (High-Impact)

| Rank | File | Lines | Why |
|------|------|-------|-----|
| ⭐ | `internal/webapp/server.go` | 570 | Central routing, all 20+ API handlers, Git ops |
| ⭐ | `internal/webapp/store.go` | 305 | Entire persistence layer for projects/requirements |
| ⭐ | `internal/webapp/models.go` | 172 | All data contracts (20+ structs) |
| ⭐ | `internal/webapp/assets.go` | 271 | S3 upload pipeline, SigV4 signing, local fallback |
| ⭐ | `web/app.js` | 1224 | Full legacy frontend SPA logic (ES6, no framework) |
| 🔶 | `internal/webapp/llm.go` | 216 | LLM proxy with model fallback chain |
| 🔶 | `internal/webapp/prompt.go` | 179 | Prompt assembly from project/requirement/editor |
| 🔶 | `internal/webapp/connectors.go` | 185 | OpenClaw & ClawHub outbound communication |
| 🔶 | `internal/webapp/documents.go` | 250 | MinerU document parsing orchestration |
| 🔶 | `internal/coordination/core.go` | 196 | Agent-task matching engine |
| 🔷 | `cmd/opclawctl/main.go` | 219 | CLI tool (9 subcommands) |
| 🔷 | `control_layer/artifacts.go` | 55 | Artifact persistence |
| 📐 | `webapps/requirement-editor/src/RequirementEditor.tsx` | 280 | Rich text React editor component |

---

## Entry Points

| Entry Point | Command | Port / Mode | Description |
|-------------|---------|-------------|-------------|
| `cmd/mutesolo-web/main.go` | `go run .` or `./mutesolo-web` | `127.0.0.1:8787` | Web console HTTP server, serves web UI + REST API |
| `cmd/opclawctl/main.go` | `opclawctl` | CLI | Coordination engine CLI: pipeline, agents, skills, tasks, events |
| `webapps/requirement-editor/src/main.tsx` | `vite dev` | Vite dev server | React SPA for rich text requirement editing |
| `web/index.html` | served via mutesolo-web | N/A | Legacy console bootstrap (loads `app.js`) |

```
Execution flow:
  mutesolo-web
    └─ main() → flag.Parse → webapp.NewServer → http.ListenAndServe(:8787)
       └─ Server.Handler() → http.NewServeMux() with 20+ routes
          └─ Each route → handlerFunc → Store.WithState → CRUD / outbound API

  opclawctl
    └─ main() → flag.Parse → dispatch subcommand
       ├─ pipeline run → control_layer.RunPipeline
       ├─ agents / skills → coordination.State
       ├─ tasks → coordination core CRUD
       └─ events → coordination state events
```

---

## Stats

- **Go files**: 31 (29 source + 2 cmd)
- **TS/TSX**: 3 files (component + entry + env types)
- **JS/CSS/HTML**: 3 files (legacy frontend)
- **Total LOC**: ~5,455
- **Go module**: Go 1.22, no external dependencies beyond stdlib
- **Frontend deps**: React, BlockNote, Vite, `@vitejs/plugin-react`
- **External services**: OpenClaw, ClawHub, OpenCode (LLM), Tailscale, S3 (optional), MinerU
