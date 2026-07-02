# OpenClaw Coordination Layer

Minimal coordination brain for OpenClaw. This repository implements decision and state only:

- Agent Registry from the A2A side, mocked locally by default.
- Skill Registry from ClawHub, mocked locally by default.
- Task to Agent matching by capability coverage.
- Task and Session state in a small JSON store.
- Simple append-only events shown through the CLI.
- Controlled generation pipeline that writes validated artifacts only.

It does not implement runtime execution, workflow orchestration, a web UI, distributed coordination, or a task platform.

## Build

```sh
go build ./cmd/opclawctl
go build ./cmd/mutesolo-web
```

## State

The CLI stores state at `.openclaw/state.json` by default. Override it with:

```sh
OPENCLAW_STATE=/tmp/openclaw-state.json opclawctl agents list
```

## CLI

```sh
opclawctl agents list
opclawctl skills list
opclawctl tasks create -id task-1 -caps code,test
opclawctl tasks match task-1
opclawctl tasks assign task-1 agent-a
opclawctl events tail
opclawctl pipeline run -prompt "write a parser helper"
```

`tasks create` also accepts capabilities as positional arguments:

```sh
opclawctl tasks create -id task-2 research summarize
```

## Matching

The matcher compares `task.required_caps` with `agent.skills`, ignores offline or busy agents, and chooses the online agent with the highest capability coverage. Ties are resolved by agent ID for deterministic behavior.

## Data Models

- `Agent`: `id`, `address`, `status`, `skills[]`
- `Skill`: `id`, `capabilities[]`, `version`
- `Task`: `id`, `required_caps[]`, `status`
- `Session`: `id`, `agent_id`, `task_id`, `status`
- `Event`: `type`, `entity_id`, `payload`, `timestamp`

## Controlled Generation

The `control_layer/` module turns generation into a bounded pipeline:

```text
input prompt -> generation -> validation -> artifact
```

Generated output is classified as:

- `safe_module_code`
- `infrastructure_code`
- `system_design_code`

System design code is blocked by default. The pipeline never writes generated code into runtime directories and never starts another generation automatically. Every run stores a deterministic JSON artifact under `artifacts/` unless another artifact directory is passed:

```sh
opclawctl pipeline run -prompt "write a string helper"
opclawctl pipeline run -prompt "rewrite system architecture" # stored but blocked
```

Use `-approve-system` only as a manual override for reviewing system-level artifacts. It still stores an artifact; it does not auto-apply code.

## Web Console

Start the local console:

```sh
go run ./cmd/mutesolo-web
```

Open `http://127.0.0.1:8787`.

### Local Object Storage

Requirement screenshots and file blocks are stored in a local MinIO bucket instead of browser-only blob URLs.

Start MinIO:

```sh
cp .env.example .env
docker compose up -d minio minio-init
```

MinIO endpoints:

- API/static assets: `http://127.0.0.1:9000`
- Console: `http://127.0.0.1:9001`
- Default bucket: `Mutesolo-assets`

The `minio-init` container creates the bucket, enables anonymous download for stored assets, and installs a lifecycle rule that expires objects after 7 days based on object creation time.

The web backend uploads files through `POST /api/assets` using these environment variables:

```sh
export MUTESOLO_MINIO_ENDPOINT=http://127.0.0.1:9000
export MUTESOLO_MINIO_PUBLIC_URL=http://127.0.0.1:9000
export MUTESOLO_MINIO_BUCKET=Mutesolo-assets
export MUTESOLO_MINIO_ACCESS_KEY=Mutesolo
export MUTESOLO_MINIO_SECRET_KEY=Mutesolo123
export MUTESOLO_ASSET_FALLBACK_DIR=.openclaw/assets
go run ./cmd/mutesolo-web
```

If MinIO is not running, the backend falls back to `.openclaw/assets` and serves files from `/assets/...` so screenshots can still render in the local Requirement Editor. The fallback directory is pruned on upload for files older than 7 days. MinIO remains the preferred storage path because its bucket lifecycle rule performs creation-time expiry automatically.

For another OpenClaw device to read previews through Tailscale, set `MUTESOLO_MINIO_PUBLIC_URL` to a Tailscale-reachable URL instead of `127.0.0.1`.

### Local Document Parsing

Uploaded documents should be parsed locally before any LLM prompt generation. Mutesolo uses MinerU as the native document intermediary:

```text
Requirement Editor upload -> MinIO/local asset -> MinerU parse -> Markdown + JSON -> backend context builder -> LLM prompt generation
```

Set up and test the local parser:

```sh
/Users/soda/.cache/codex-runtimes/codex-primary-runtime/dependencies/python/bin/python3 -m venv .venv-mineru
.venv-mineru/bin/python -m pip install -U pip
.venv-mineru/bin/python -m pip install -r requirements-mineru.txt
.venv-mineru/bin/mineru-models-download -m pipeline -s modelscope
scripts/mineru-parse path/to/document.pdf
```

See `docs/mineru-native.md` for the full native setup, output contract, and smoke-test notes.

Parse an uploaded asset through the backend:

```sh
curl -X POST http://127.0.0.1:8787/api/documents/parse \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "requirement.png",
    "url": "http://127.0.0.1:9000/Mutesolo-assets/2026/06/25/asset.png",
    "storageKey": "2026/06/25/asset.png",
    "source": "minio",
    "method": "ocr"
  }'
```

The response includes `markdown` and `contentList`. Local MinerU paths stay server-side so they are not accidentally forwarded to an online LLM. For local debugging only, set `MUTESOLO_ALLOW_PARSE_PATHS=1` and pass `path`; production UI flows should upload the file first and parse the returned asset object.

The console provides:

- OpenClaw online/offline probe through the configured Tailscale URL by reading `/.well-known/agent-card.json`.
- Manual prompt delivery to OpenClaw through the configured A2A gateway URL.
- Discord handoff flow for IM-based OpenClaw agents: copy a Discord-ready prompt, open the configured Discord channel or DM, and send it manually.
- Optional Discord server widget preview and OpenClaw bot mention support using `<@BOT_ID>`.
- GitHub repository field and a guarded push action that refuses to push while local changes are uncommitted.
- Private ClawHub skill listing and skill detail pages through the configured private ClawHub URL.
- Controlled skill install requests sent to a selected OpenClaw through Tailscale/A2A as an instruction message.
- Project list, project board, and task detail pages.
- Task board closure by pasting OpenClaw A's GitHub commit SHA and closing selected requirements in bulk.
- Coordination prompt generation from project plan, requirement document, and selected requirement.
- Segmented prompt output stored as a controlled artifact under `artifacts/`.
- Runtime descriptors for Go, Node/TypeScript, and Python plugin compatibility.

The web layer does not execute generated code, does not auto-change system architecture, and does not trigger recursive generation. LLM optimization, plugin execution, and OpenClaw delivery remain explicit integration points bounded by the control layer and artifact storage.

Discord is intentionally handled as a human-in-the-loop IM handoff. The server widget can be embedded for presence and navigation, but it is not treated as an authenticated chat input. The console prepares the message, optionally prefixes it with `<@BOT_ID>`, copies it to the clipboard, and opens the configured Discord URL. OpenClaw A is expected to commit results to GitHub and reply with `commit: <sha>`; Mutesolo records that commit ID when you close selected requirement points on the task board.

The UI uses a workspace structure inspired by Huly-style product consoles: a narrow app rail, module sidebar, project list, board, task detail, private ClawHub, runtime descriptors, and connections page. It intentionally avoids the heavier human-team workflow model because Mutesolo coordinates AI agents and bounded artifacts rather than personnel. In this mapping, a Huly-style person becomes an OpenClaw instance, and a Huly-style issue becomes a requirement point assigned to an online OpenClaw.
# Mutesolo
