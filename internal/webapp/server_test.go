package webapp

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestLLMTestRouteIsRegistered(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state.json"))
	server := NewServer(store, "web")
	request := httptest.NewRequest(http.MethodPost, "/api/llm/test", strings.NewReader(`{"llm":{}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("/api/llm/test returned 404; route is not registered")
	}
}

func TestDocumentParseRouteIsRegistered(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "state.json"))
	server := NewServer(store, "web")
	request := httptest.NewRequest(http.MethodPost, "/api/documents/parse", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code == http.StatusNotFound {
		t.Fatal("/api/documents/parse returned 404; route is not registered")
	}
}
