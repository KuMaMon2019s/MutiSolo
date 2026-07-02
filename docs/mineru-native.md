# MinerU Native Document Intermediary

Mutesolo should not send raw uploaded files, localhost URLs, or browser blob URLs directly to an online LLM. The local pipeline is:

```text
Requirement Editor upload -> MinIO/local asset -> MinerU parse -> Markdown + JSON -> backend context builder -> LLM prompt generation
```

MinerU is the document intermediary. It converts PDFs, images, DOCX, PPTX, and XLSX into readable Markdown and structured JSON that the backend can safely package for an LLM.

## Installed Runtime

This workspace uses a native Python virtual environment:

```sh
.venv-mineru/bin/python --version
.venv-mineru/bin/mineru --version
```

Current verified package set:

```text
mineru 3.4.0
torch 2.12.1
torchvision 0.27.1
transformers 4.57.6
shapely 2.1.2
six 1.17.0
pyclipper 1.4.0
```

The system Python on this machine is too old for MinerU, so the venv uses the bundled Python 3.12 runtime.

## Recreate The Environment

```sh
/Users/soda/.cache/codex-runtimes/codex-primary-runtime/dependencies/python/bin/python3 -m venv .venv-mineru
.venv-mineru/bin/python -m pip install -U pip
.venv-mineru/bin/python -m pip install -r requirements-mineru.txt
.venv-mineru/bin/mineru-models-download -m pipeline -s modelscope
```

The model downloader writes MinerU config to `/Users/soda/mineru.json` and stores pipeline models under the local ModelScope cache.

## Parse A File

Use the wrapper script from the repository root:

```sh
scripts/mineru-parse path/to/input.pdf
scripts/mineru-parse path/to/screenshot.png .openclaw/mineru-output
```

Defaults:

- `MINERU_BACKEND=pipeline`
- `MINERU_METHOD=auto`
- `MINERU_FORMULA=false`
- output directory `.openclaw/mineru`

Formula parsing is disabled by default for the local wrapper because the first Mutesolo use case is requirement text, screenshots, and ordinary documents. Enable it per command when math extraction matters:

```sh
MINERU_FORMULA=true scripts/mineru-parse path/to/math.pdf
```

## Output Contract

MinerU writes one folder per input document. The backend should read these files first:

- `*_content_list.json`: structured content blocks suitable for a context builder.
- `*_content_list_v2.json`: richer structured blocks where available.
- `*.md`: readable Markdown fallback.
- `images/`: extracted images referenced by the Markdown/JSON.

The LLM should receive backend-curated text and image references only after this parsing step. It should not receive local filesystem paths, browser blob URLs, or raw `localhost` URLs.

## Optional Local API

For a long-running parser service:

```sh
.venv-mineru/bin/mineru-api --host 127.0.0.1 --port 8788
```

Then call the wrapper with:

```sh
MINERU_API_URL=http://127.0.0.1:8788 scripts/mineru-parse path/to/input.pdf
```

Keep the API bound to localhost unless there is an explicit Tailscale access plan.

## Verified Smoke Test

This command successfully parsed a pasted screenshot through OCR:

```sh
scripts/mineru-parse /var/folders/rx/26jrr8v15lq416cvmkykm_w00000gn/T/codex-clipboard-323a26f6-1e23-4308-babb-6e8e89dcecc4.png .openclaw/mineru-smoke-script -m ocr
```

Verified outputs include Markdown, content JSON, middle/model JSON, layout PDFs, and extracted image assets.
