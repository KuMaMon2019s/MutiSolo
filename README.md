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
# MutiSolo
