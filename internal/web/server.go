package web

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/mahmoud-nn/devlaunch/internal/runtime"
)

type Server struct {
	httpServer *http.Server
}

type pageProject struct {
	ID       string
	Name     string
	RootPath string
	Status   string
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	server := &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}

	mux.HandleFunc("/", server.handleIndex)
	mux.HandleFunc("/projects", server.handleProjects)
	mux.HandleFunc("/projects/", server.handleProjectAction)
	mux.HandleFunc("/__control/stop", server.handleStop)
	return server
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	records, err := runtime.ListProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projects := make([]pageProject, 0, len(records))
	for _, record := range records {
		projects = append(projects, pageProject{
			ID:       record.ID,
			Name:     record.Name,
			RootPath: record.RootPath,
			Status:   string(record.LastKnownStatus),
		})
	}

	tpl := template.Must(template.New("index").Parse(indexHTML))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.Execute(w, map[string]any{"Projects": projects})
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	records, err := runtime.ListProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleProjectAction(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path)
	if len(parts) < 3 || parts[0] != "projects" {
		http.NotFound(w, r)
		return
	}

	target, err := runtime.ResolveTarget(parts[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	switch {
	case r.Method == http.MethodGet && parts[2] == "status":
		status, err := runtime.Status(target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, status)
	case r.Method == http.MethodPost && parts[2] == "start":
		options, err := decodeExecutionOptions(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		status, err := runtime.Start(target, options)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, status)
	case r.Method == http.MethodPost && parts[2] == "stop":
		options, err := decodeExecutionOptions(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		status, err := runtime.Stop(target, options)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, status)
	case r.Method == http.MethodPost && parts[2] == "open-folder":
		if err := runtime.OpenProjectFolder(target); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = s.Shutdown(context.Background())
	}()
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeExecutionOptions(r *http.Request) (runtime.ExecutionOptions, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return runtime.ExecutionOptions{Interactive: false}, nil
	}
	defer r.Body.Close()
	var options runtime.ExecutionOptions
	if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
		return runtime.ExecutionOptions{}, err
	}
	options.Interactive = false
	if options.Decisions == nil {
		options.Decisions = map[string]bool{}
	}
	return options, nil
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>devlaunch</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script>
    async function act(id, action) {
      const response = await fetch('/projects/' + id + '/' + action, { method: 'POST' });
      if (!response.ok) {
        const text = await response.text();
        alert(text);
        return;
      }
      window.location.reload();
    }
  </script>
</head>
<body class="min-h-screen bg-slate-100 text-slate-900">
  <main class="mx-auto max-w-6xl px-6 py-10">
    <header class="mb-8">
      <div class="inline-flex rounded-full bg-teal-100 px-3 py-1 text-sm font-semibold text-teal-800">localhost control plane</div>
      <h1 class="mt-4 text-5xl font-black tracking-tight">devlaunch</h1>
      <p class="mt-2 max-w-2xl text-slate-600">Simple local launcher for your registered projects. The UI is only a convenience layer over the same runtime used by the CLI.</p>
    </header>
    <section class="grid gap-4">
      {{if .Projects}}
        {{range .Projects}}
        <article class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
          <div class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div class="mb-3 inline-flex rounded-full px-3 py-1 text-xs font-bold uppercase tracking-[0.2em] {{if eq .Status "running"}}bg-emerald-100 text-emerald-700{{else if eq .Status "stopped"}}bg-slate-200 text-slate-700{{else}}bg-amber-100 text-amber-700{{end}}">{{.Status}}</div>
              <h2 class="text-2xl font-bold">{{.Name}}</h2>
              <p class="mt-2 break-all text-sm text-slate-500">{{.RootPath}}</p>
            </div>
            <div class="flex flex-wrap gap-3">
              <button class="rounded-full bg-teal-700 px-4 py-2 text-sm font-semibold text-white" onclick="act('{{.ID}}','start')">Start</button>
              <button class="rounded-full bg-rose-700 px-4 py-2 text-sm font-semibold text-white" onclick="act('{{.ID}}','stop')">Stop</button>
              <button class="rounded-full bg-slate-900 px-4 py-2 text-sm font-semibold text-white" onclick="act('{{.ID}}','open-folder')">Open Folder</button>
            </div>
          </div>
        </article>
        {{end}}
      {{else}}
        <article class="rounded-3xl border border-dashed border-slate-300 bg-white/70 p-10 text-center text-slate-500">
          No projects registered yet. Run <code class="rounded bg-slate-100 px-2 py-1 text-slate-800">devlaunch init</code> inside a repo first.
        </article>
      {{end}}
    </section>
  </main>
</body>
</html>`
