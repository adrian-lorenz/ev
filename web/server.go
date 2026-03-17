package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"

	"envault/vault"
)

//go:embed templates
var templateFiles embed.FS

//go:embed static
var staticFiles embed.FS

// Server runs the HTMX management UI.
type Server struct {
	Vault     *vault.Vault
	VaultPath string
	Password  string
	Port      int
	Bind      string

	srv  *http.Server
	tmpl *template.Template
}

type pageData struct {
	Projects []string
}

// SecretAreaData is passed to the secret-area and secret-rows templates.
type SecretAreaData struct {
	Project string
	Secrets []SecretRow
}

// SecretRow holds a key name for the secrets table.
type SecretRow struct {
	Key string
}

// RevealData is passed to reveal-value and edit-row templates.
type RevealData struct {
	Project string
	Key     string
	Value   string
}

// Start builds the mux and begins serving.
func (s *Server) Start() error {
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

	var err error
	s.tmpl, err = template.New("").Funcs(funcs).ParseFS(templateFiles, "templates/index.html")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServerFS(staticFiles))
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /projects/{project}/secrets", s.handleSecretList)
	mux.HandleFunc("POST /projects/{project}/secrets", s.handleSetSecret)
	mux.HandleFunc("DELETE /projects/{project}/secrets/{key}", s.handleDeleteSecret)
	mux.HandleFunc("GET /projects/{project}/secrets/{key}/reveal", s.handleRevealSecret)
	mux.HandleFunc("GET /projects/{project}/secrets/{key}/row", s.handleSecretRow)
	mux.HandleFunc("GET /projects/{project}/secrets/{key}/edit", s.handleEditSecret)
	mux.HandleFunc("PUT /projects/{project}/secrets/{key}", s.handleUpdateSecret)
	mux.HandleFunc("POST /projects", s.handleCreateProject)
	mux.HandleFunc("DELETE /projects/{project}", s.handleDeleteProject)

	addr := fmt.Sprintf("%s:%d", s.Bind, s.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}

	s.srv = &http.Server{Handler: s.csrfMiddleware(mux)}
	return s.srv.Serve(ln)
}

func (s *Server) Shutdown(ctx context.Context) {
	if s.srv != nil {
		s.srv.Shutdown(ctx)
	}
}

// csrfMiddleware rejects mutating requests from unexpected origins.
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
			origin := r.Header.Get("Origin")
			allowed := fmt.Sprintf("http://localhost:%d", s.Port)
			if origin != "" && origin != allowed {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) save() error {
	return s.Vault.Save(s.VaultPath, s.Password)
}

func (s *Server) refreshSession(project string) {
	vars := s.Vault.GetAll(project)
	_ = vault.RefreshSession(project, vars)
}

func (s *Server) buildSecretArea(project string) SecretAreaData {
	keys := s.Vault.ListKeys(project)
	rows := make([]SecretRow, len(keys))
	for i, k := range keys {
		rows[i] = SecretRow{Key: k}
	}
	return SecretAreaData{Project: project, Secrets: rows}
}

// --- Handlers ---

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.render(w, "page", pageData{Projects: s.Vault.ListProjects()})
}

func (s *Server) handleSecretList(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	s.render(w, "secret-area", s.buildSecretArea(project))
}

func (s *Server) handleSetSecret(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	key := strings.ToUpper(strings.TrimSpace(r.FormValue("key")))
	value := r.FormValue("value")

	if err := vault.ValidateName(key); err != nil {
		http.Error(w, "invalid key: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.Vault.Set(project, key, value)
	if err := s.save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.refreshSession(project)
	vault.Audit("set-secret", "project="+vault.HashName(project)+" key="+vault.HashName(key))
	s.render(w, "secret-rows", s.buildSecretArea(project))
}

func (s *Server) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	key := r.PathValue("key")

	if err := vault.ValidateName(project); err != nil {
		http.Error(w, "invalid project name", http.StatusBadRequest)
		return
	}
	if err := vault.ValidateName(key); err != nil {
		http.Error(w, "invalid key name", http.StatusBadRequest)
		return
	}

	if err := s.Vault.Delete(project, key); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := s.save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.refreshSession(project)
	vault.Audit("delete-secret", "project="+vault.HashName(project)+" key="+vault.HashName(key))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRevealSecret(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	key := r.PathValue("key")

	value, ok := s.Vault.Get(project, key)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	s.render(w, "reveal-value", RevealData{Key: key, Value: value})
}

func (s *Server) handleSecretRow(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	key := r.PathValue("key")
	s.render(w, "secret-row", struct {
		Project string
		Key     string
	}{project, key})
}

func (s *Server) handleEditSecret(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	key := r.PathValue("key")

	value, ok := s.Vault.Get(project, key)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	s.render(w, "edit-row", RevealData{Key: key, Value: value, Project: project})
}

func (s *Server) handleUpdateSecret(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")
	key := r.PathValue("key")
	value := r.FormValue("value")

	if err := vault.ValidateName(project); err != nil {
		http.Error(w, "invalid project name", http.StatusBadRequest)
		return
	}
	if err := vault.ValidateName(key); err != nil {
		http.Error(w, "invalid key name", http.StatusBadRequest)
		return
	}

	s.Vault.Set(project, key, value)
	if err := s.save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.refreshSession(project)
	vault.Audit("update-secret", "project="+vault.HashName(project)+" key="+vault.HashName(key))
	s.render(w, "secret-row", struct {
		Project string
		Key     string
	}{project, key})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if err := vault.ValidateName(name); err != nil {
		http.Error(w, "invalid project name: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.Vault.CreateProject(name); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := s.save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vault.Audit("create-project", "project="+vault.HashName(name))
	s.render(w, "project-list", s.Vault.ListProjects())
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	project := r.PathValue("project")

	if err := vault.ValidateName(project); err != nil {
		http.Error(w, "invalid project name", http.StatusBadRequest)
		return
	}

	if err := s.Vault.DeleteProject(project); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := s.save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vault.Audit("delete-project", "project="+vault.HashName(project))
	// Return updated project list; HTMX OOB clears the secret area
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.tmpl.ExecuteTemplate(w, "project-list", s.Vault.ListProjects())
	fmt.Fprintf(w, `<div id="secret-area" hx-swap-oob="true"><div class="p-8"><p class="text-sm text-gray-600">← Select a project to view secrets</p></div></div>`)
}
