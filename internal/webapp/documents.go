package webapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const maxDocumentDownloadBytes = 64 << 20

type DocumentParseRequest struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	StorageKey string `json:"storageKey"`
	Source     string `json:"source"`
	Path       string `json:"path"`
	Method     string `json:"method"`
	Backend    string `json:"backend"`
}

type DocumentParseResponse struct {
	Name            string          `json:"name"`
	InputPath       string          `json:"-"`
	OutputDir       string          `json:"-"`
	MarkdownPath    string          `json:"-"`
	ContentListPath string          `json:"-"`
	Markdown        string          `json:"markdown,omitempty"`
	ContentList     json.RawMessage `json:"contentList,omitempty"`
	DurationMS      int64           `json:"durationMs"`
}

func (s Server) handleDocumentParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input DocumentParseRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	result, err := ParseDocument(ctx, input)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, result)
}

func ParseDocument(ctx context.Context, input DocumentParseRequest) (DocumentParseResponse, error) {
	started := time.Now()
	localPath, displayName, err := prepareDocumentInput(ctx, input)
	if err != nil {
		return DocumentParseResponse{}, err
	}
	outputDir := envOrDefault("MUTESOLO_MINERU_OUTPUT_DIR", ".openclaw/mineru")
	method := strings.TrimSpace(input.Method)
	if method == "" {
		method = "auto"
	}
	backend := strings.TrimSpace(input.Backend)
	if backend == "" {
		backend = "pipeline"
	}

	script := envOrDefault("MUTESOLO_MINERU_SCRIPT", filepath.Join("scripts", "mineru-parse"))
	cmd := exec.CommandContext(ctx, script, localPath, outputDir)
	cmd.Env = append(os.Environ(), "MINERU_BACKEND="+backend, "MINERU_METHOD="+method)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return DocumentParseResponse{}, fmt.Errorf("mineru parse failed: %w: %s", err, strings.TrimSpace(string(data)))
	}

	markdownPath, contentListPath, err := findMinerUOutputs(outputDir, localPath)
	if err != nil {
		return DocumentParseResponse{}, err
	}
	response := DocumentParseResponse{
		Name:            displayName,
		InputPath:       localPath,
		OutputDir:       outputDir,
		MarkdownPath:    markdownPath,
		ContentListPath: contentListPath,
		DurationMS:      time.Since(started).Milliseconds(),
	}
	if markdownPath != "" {
		response.Markdown = readSmallText(markdownPath, 256<<10)
	}
	if contentListPath != "" {
		if raw, err := os.ReadFile(contentListPath); err == nil {
			response.ContentList = json.RawMessage(raw)
		}
	}
	return response, nil
}

func prepareDocumentInput(ctx context.Context, input DocumentParseRequest) (string, string, error) {
	if strings.TrimSpace(input.Path) != "" {
		if envOrDefault("MUTESOLO_ALLOW_PARSE_PATHS", "") != "1" {
			return "", "", fmt.Errorf("path parsing is disabled; upload the file first or set MUTESOLO_ALLOW_PARSE_PATHS=1 for local debugging")
		}
		cleaned := filepath.Clean(input.Path)
		if _, err := os.Stat(cleaned); err != nil {
			return "", "", fmt.Errorf("stat input path: %w", err)
		}
		return cleaned, filepath.Base(cleaned), nil
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = path.Base(strings.TrimSpace(input.StorageKey))
	}
	if name == "" || name == "." || name == "/" {
		name = "document.bin"
	}

	if strings.TrimSpace(input.Source) == "local_static_fallback" && strings.TrimSpace(input.StorageKey) != "" {
		localDir := envOrDefault("MUTESOLO_ASSET_FALLBACK_DIR", ".openclaw/assets")
		localPath := filepath.Join(localDir, filepath.FromSlash(input.StorageKey))
		if _, err := os.Stat(localPath); err != nil {
			return "", "", fmt.Errorf("stat local asset: %w", err)
		}
		return localPath, name, nil
	}

	if strings.TrimSpace(input.URL) == "" {
		return "", "", fmt.Errorf("document url, storageKey, or path is required")
	}
	parsed, err := url.Parse(input.URL)
	if err != nil {
		return "", "", fmt.Errorf("parse document url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("only http(s) document urls are supported")
	}
	return downloadDocument(ctx, input.URL, name)
}

func downloadDocument(ctx context.Context, rawURL string, name string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("download document: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("download document failed: status %d", resp.StatusCode)
	}
	safeName := safeFilename(name)
	if safeName == "" {
		safeName = "document.bin"
	}
	dir := filepath.Join(envOrDefault("MUTESOLO_MINERU_INPUT_DIR", ".openclaw/mineru-inputs"), time.Now().UTC().Format("20060102"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	target := filepath.Join(dir, randomHex(8)+"-"+safeName)
	file, err := os.Create(target)
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	limited := io.LimitReader(resp.Body, maxDocumentDownloadBytes+1)
	written, err := io.Copy(file, limited)
	if err != nil {
		return "", "", fmt.Errorf("write downloaded document: %w", err)
	}
	if written > maxDocumentDownloadBytes {
		return "", "", fmt.Errorf("document exceeds %d bytes", maxDocumentDownloadBytes)
	}
	return target, name, nil
}

func findMinerUOutputs(outputDir string, inputPath string) (string, string, error) {
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	var markdownPath string
	var contentListPath string
	err := filepath.WalkDir(outputDir, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if !strings.Contains(current, base) {
			return nil
		}
		if markdownPath == "" && strings.HasSuffix(name, ".md") {
			markdownPath = current
		}
		if contentListPath == "" && strings.HasSuffix(name, "_content_list.json") {
			contentListPath = current
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}
	if markdownPath == "" && contentListPath == "" {
		return "", "", fmt.Errorf("mineru output not found under %s", outputDir)
	}
	return markdownPath, contentListPath, nil
}

func readSmallText(filePath string, maxBytes int64) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes))
	if err != nil {
		return ""
	}
	return string(data)
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '.', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, name)
	return strings.Trim(name, ".-")
}
