package web

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"envault/vault"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	v, err := vault.Open(filepath.Join(t.TempDir(), "missing-vault.json"), "test-password")
	if err != nil {
		t.Fatalf("vault.Open failed: %v", err)
	}

	funcs := template.FuncMap{
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			m := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				k, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				m[k] = pairs[i+1]
			}
			return m, nil
		},
	}
	tmpl, err := template.New("").Funcs(funcs).ParseFS(templateFiles, "templates/index.html")
	if err != nil {
		t.Fatalf("ParseFS failed: %v", err)
	}

	return &Server{
		Vault:     v,
		VaultPath: filepath.Join(t.TempDir(), "vault.json"),
		Password:  "test-password",
		tmpl:      tmpl,
	}
}

func TestHandleCreateProjectReturnsConflictForDuplicate(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.Vault.CreateProject("demo"); err != nil {
		t.Fatalf("CreateProject setup failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/projects", nil)
	req.Form = map[string][]string{"name": {"demo"}}
	rec := httptest.NewRecorder()

	srv.handleCreateProject(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestHandleDeleteSecretReturnsNotFoundForMissingKey(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.Vault.CreateProject("demo"); err != nil {
		t.Fatalf("CreateProject setup failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/projects/demo/secrets/MISSING", nil)
	req.SetPathValue("project", "demo")
	req.SetPathValue("key", "MISSING")
	rec := httptest.NewRecorder()

	srv.handleDeleteSecret(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleDeleteProjectReturnsNotFoundForMissingProject(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/projects/missing", nil)
	req.SetPathValue("project", "missing")
	rec := httptest.NewRecorder()

	srv.handleDeleteProject(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
