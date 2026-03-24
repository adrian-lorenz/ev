package web

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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

// --- CSRF middleware ---

func TestCSRFMiddleware_RejectsWrongOrigin(t *testing.T) {
	srv := newTestServer(t)
	srv.Port = 9871

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := srv.csrfMiddleware(inner)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		called = false
		req := httptest.NewRequest(method, "/", nil)
		req.Header.Set("Origin", "http://evil.example.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("%s with wrong origin: got %d, want 403", method, rec.Code)
		}
		if called {
			t.Errorf("%s with wrong origin should not reach inner handler", method)
		}
	}
}

func TestCSRFMiddleware_AllowsCorrectOrigin(t *testing.T) {
	srv := newTestServer(t)
	srv.Port = 9872

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := srv.csrfMiddleware(inner)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "http://localhost:9872")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("correct origin: got %d, want 200", rec.Code)
	}
}

func TestCSRFMiddleware_AllowsNoOrigin(t *testing.T) {
	srv := newTestServer(t)
	srv.Port = 9873

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := srv.csrfMiddleware(inner)

	// No Origin header (e.g. curl, direct form submit) — should be allowed
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("no origin: got %d, want 200", rec.Code)
	}
}

func TestCSRFMiddleware_GETNotChecked(t *testing.T) {
	srv := newTestServer(t)
	srv.Port = 9874

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := srv.csrfMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET with wrong origin should not be blocked: got %d", rec.Code)
	}
}

// --- handleSetSecret ---

func TestHandleSetSecretRejectsBadKey(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.Vault.CreateProject("demo"); err != nil {
		t.Fatal(err)
	}

	for _, badKey := range []string{"", "KEY WITH SPACE", "KEY=BAD"} {
		req := httptest.NewRequest(http.MethodPost, "/projects/demo/secrets", nil)
		req.SetPathValue("project", "demo")
		req.Form = map[string][]string{"key": {badKey}, "value": {"v"}}
		rec := httptest.NewRecorder()

		srv.handleSetSecret(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("key %q: got %d, want 400", badKey, rec.Code)
		}
	}
}

// --- handleRevealSecret ---

func TestHandleRevealSecretReturnsValueForExistingKey(t *testing.T) {
	srv := newTestServer(t)
	srv.Vault.CreateProject("demo")
	srv.Vault.Set("demo", "MY_KEY", "supersecret")

	req := httptest.NewRequest(http.MethodGet, "/projects/demo/secrets/MY_KEY/reveal", nil)
	req.SetPathValue("project", "demo")
	req.SetPathValue("key", "MY_KEY")
	rec := httptest.NewRecorder()

	srv.handleRevealSecret(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "supersecret") {
		t.Error("response should contain the secret value")
	}
}

func TestHandleRevealSecretReturnsNotFoundForMissingKey(t *testing.T) {
	srv := newTestServer(t)
	srv.Vault.CreateProject("demo")

	req := httptest.NewRequest(http.MethodGet, "/projects/demo/secrets/MISSING/reveal", nil)
	req.SetPathValue("project", "demo")
	req.SetPathValue("key", "MISSING")
	rec := httptest.NewRecorder()

	srv.handleRevealSecret(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- handleUpdateSecret ---

func TestHandleUpdateSecretRejectsBadProjectOrKey(t *testing.T) {
	srv := newTestServer(t)

	cases := []struct {
		project string
		key     string
	}{
		{"bad project", "KEY"},
		{"demo", "bad key"},
		{"", "KEY"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPut, "/", nil)
		req.SetPathValue("project", tc.project)
		req.SetPathValue("key", tc.key)
		req.Form = map[string][]string{"value": {"v"}}
		rec := httptest.NewRecorder()

		srv.handleUpdateSecret(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("project=%q key=%q: got %d, want 400", tc.project, tc.key, rec.Code)
		}
	}
}
