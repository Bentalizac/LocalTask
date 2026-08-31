package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"localtask/internal/git"
	"localtask/internal/project"
	"localtask/internal/registry"
	"localtask/internal/ticket"
)

// newTestServer loads a fresh copy of testdata/sample-project as "DanceNote"
// and returns a Server plus its root dir (for reload-and-verify assertions).
func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "sample-project")
	root := t.TempDir()

	data, err := os.ReadFile(filepath.Join(src, "taskboard.json"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "taskboard.json"), data, 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "video-processing.md"), []byte("# Video Processing\n"), 0o644); err != nil {
		t.Fatalf("writing doc: %v", err)
	}

	p, err := project.LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	reg := registry.NewRegistry(nil, map[string]*project.Project{"DanceNote": p}, nil)
	return New(reg, ""), root
}

func doRequest(t *testing.T, s *Server, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestListProjects(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var resp []projectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].Name != "DanceNote" || resp[0].Metrics.Total != 4 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects/Nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body)
	}
}

func TestListTicketsWithFilter(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects/DanceNote/tickets?status=done", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var tickets []ticket.Ticket
	if err := json.Unmarshal(rec.Body.Bytes(), &tickets); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tickets) != 2 {
		t.Fatalf("expected 2 done tickets, got %d", len(tickets))
	}
}

func TestListTicketsInvalidFilter(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects/DanceNote/tickets?status=bogus", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects/DanceNote/tickets/NOPE-1", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body)
	}
}

func TestCreateTicketPersists(t *testing.T) {
	s, root := newTestServer(t)
	rec := doRequest(t, s, http.MethodPost, "/api/projects/DanceNote/tickets", map[string]any{
		"title": "Add retry handling",
		"tags":  []string{"backend"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body)
	}
	var created ticket.Ticket
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.ID != "DN-43" {
		t.Fatalf("expected generated id DN-43, got %s", created.ID)
	}

	reloaded, err := project.LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, err := reloaded.GetTicket("DN-43"); err != nil {
		t.Fatalf("expected ticket to be persisted: %v", err)
	}
}

func TestCreateTicketMissingTitle(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodPost, "/api/projects/DanceNote/tickets", map[string]any{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body)
	}
}

func TestUpdateTicketPersists(t *testing.T) {
	s, root := newTestServer(t)
	rec := doRequest(t, s, http.MethodPatch, "/api/projects/DanceNote/tickets/DN-42", map[string]any{
		"status": "review",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}

	reloaded, err := project.LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, err := reloaded.GetTicket("DN-42")
	if err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if got.Status != ticket.StatusReview {
		t.Fatalf("expected status review, got %s", got.Status)
	}
}

func TestUpdateTicketInvalidStatus(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodPatch, "/api/projects/DanceNote/tickets/DN-42", map[string]any{
		"status": "not-a-status",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body)
	}
}

func TestGetDoc(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects/DanceNote/docs/docs/video-processing.md", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var doc struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Content == "" {
		t.Fatal("expected non-empty doc content")
	}
}

func TestGetDocRejectsPathEscape(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects/DanceNote/docs/..%2F..%2Fetc%2Fpasswd", nil)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 400 or 404 for path escape, got %d: %s", rec.Code, rec.Body)
	}
}

func TestGetGitNotARepo(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects/DanceNote/git", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var info git.Info
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if info.IsRepo {
		t.Fatal("expected is_repo=false for a plain temp dir")
	}
}

func TestListDocs(t *testing.T) {
	s, _ := newTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/projects/DanceNote/docs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var docs []string
	if err := json.Unmarshal(rec.Body.Bytes(), &docs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(docs) != 1 || docs[0] != "docs/video-processing.md" {
		t.Fatalf("unexpected docs list: %+v", docs)
	}
}
