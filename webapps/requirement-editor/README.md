# Mutesolo Requirement Editor

BlockNote-based local requirement editor for Mutesolo.

## Run

```bash
cd webapps/requirement-editor
npm install
npm run dev
```

Open `http://127.0.0.1:5174/apps/requirement-editor/`.

The Vite dev server proxies `/api/*` to the local Mutesolo Go server at `http://127.0.0.1:8787`.

## Build for the Go web console

```bash
cd webapps/requirement-editor
npm run build
cd ../..
./Mutesolo-web -addr 127.0.0.1:8787 -static web
```

Open `http://127.0.0.1:8787/apps/requirement-editor/`, or use the `Editor` button in Task Detail.
