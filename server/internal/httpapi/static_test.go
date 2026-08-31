package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"localtask/internal/project"
	"localtask/internal/registry"
)

func newTestServerWithStatic(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "taskboard.json"), []byte(`{"version":1,"project":"DanceNote","tickets":[]}`), 0o644); err != nil {
		t.Fatalf("writing taskboard: %v", err)
	}
	p, err := project.LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>app shell</html>"), 0o644); err != nil {
		t.Fatalf("writing index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "app.js"), []byte("console.log('hi')"), 0o644); err != nil {
		t.Fatalf("writing app.js: %v", err)
	}

	reg := registry.NewRegistry(nil, map[string]*project.Project{"DanceNote": p}, nil)
	return New(reg, staticDir)
}

func TestStaticServesIndexAtRoot(t *testing.T) {
	s := newTestServerWithStatic(t)
	rec := doRequest(t, s, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "<html>app shell</html>" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestStaticServesRealAsset(t *testing.T) {
	s := newTestServerWithStatic(t)
	rec := doRequest(t, s, http.MethodGet, "/app.js", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "console.log('hi')" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestStaticFallsBackToIndexForClientRoutes(t *testing.T) {
	s := newTestServerWithStatic(t)
	rec := doRequest(t, s, http.MethodGet, "/projects/DanceNote/board", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "<html>app shell</html>" {
		t.Fatalf("expected SPA fallback to index.html, got: %s", rec.Body.String())
	}
}

func TestStaticDoesNotShadowAPIRoutes(t *testing.T) {
	s := newTestServerWithStatic(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON API response, got content-type %q body %q", rec.Header().Get("Content-Type"), rec.Body.String())
	}
}
