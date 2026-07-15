package web

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"radio/internal/scanner"
	"radio/internal/station"
)

//go:embed templates/*.html
var templateFS embed.FS

type Handler struct {
	mgr      *station.Manager
	port     int
	musicDir string
	tmpl     *template.Template
}

func NewHandler(mgr *station.Manager, port int, musicDir string) *Handler {
	tmpl := template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))
	return &Handler{
		mgr:      mgr,
		port:     port,
		musicDir: musicDir,
		tmpl:     tmpl,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/" && r.Method == "GET":
		h.dashboard(w, r)
	case path == "/stations" && r.Method == "POST":
		h.createStation(w, r)
	case path == "/library" && r.Method == "GET":
		h.library(w, r)
	case strings.HasPrefix(path, "/stream/"):
		h.handleStream(w, r)
	case strings.HasPrefix(path, "/stations/"):
		h.routeStation(w, r, path)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) routeStation(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	name := parts[1]

	if len(parts) == 2 && r.Method == "GET" {
		h.stationDetail(w, r, name)
		return
	}
	if len(parts) == 3 && parts[2] == "status" && r.Method == "GET" {
		h.stationStatus(w, r, name)
		return
	}
	if len(parts) == 3 && parts[2] == "sources" && r.Method == "POST" {
		h.addSource(w, r, name)
		return
	}
	if len(parts) == 4 && parts[2] == "sources" && r.Method == "DELETE" {
		index, err := strconv.Atoi(parts[3])
		if err != nil {
			http.Error(w, "invalid source index", http.StatusBadRequest)
			return
		}
		h.removeSource(w, r, name, index)
		return
	}
	http.NotFound(w, r)
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Stations": h.mgr.StationList(),
		"Port":     h.port,
		"Template": "dashboard",
	}
	h.tmpl.ExecuteTemplate(w, "base.html", data)
}

func (h *Handler) createStation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	mount := r.FormValue("mount")
	if name == "" || mount == "" {
		http.Error(w, "name and mount required", http.StatusBadRequest)
		return
	}
	if err := h.mgr.CreateStation(name, mount); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	// Redirect to dashboard to show the new station
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request) {
	mount := strings.TrimPrefix(r.URL.Path, "/stream/")
	s := h.mgr.StreamerForMount(mount)
	if s == nil {
		http.NotFound(w, r)
		return
	}
	s.ServeHTTP(w, r)
}

func (h *Handler) stationDetail(w http.ResponseWriter, r *http.Request, name string) {
	st := h.mgr.Find(name)
	if st == nil {
		http.NotFound(w, r)
		return
	}
	data := map[string]any{
		"Station":  st,
		"Port":     h.port,
		"Template": "station",
	}
	h.tmpl.ExecuteTemplate(w, "base.html", data)
}

func (h *Handler) stationStatus(w http.ResponseWriter, r *http.Request, name string) {
	st := h.mgr.Find(name)
	if st == nil {
		w.Write([]byte("offline"))
		return
	}
	fmt.Fprintf(w, "%d tracks", st.TrackCount())
}

func (h *Handler) library(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	tracks, _ := scanner.Scan(h.musicDir)

	if query != "" {
		var filtered []scanner.Track
		lower := strings.ToLower(query)
		for _, t := range tracks {
			if strings.Contains(strings.ToLower(t.Name), lower) ||
				strings.Contains(strings.ToLower(t.Path), lower) {
				filtered = append(filtered, t)
			}
		}
		tracks = filtered
	}

	data := map[string]any{
		"Tracks":   tracks,
		"Query":    query,
		"Stations": h.mgr.StationList(),
	}

	data["Template"] = "library"
	if query != "" {
		h.tmpl.ExecuteTemplate(w, "library-results", data)
		return
	}

	h.tmpl.ExecuteTemplate(w, "base.html", data)
}

func (h *Handler) addSource(w http.ResponseWriter, r *http.Request, name string) {
	st := h.mgr.Find(name)
	if st == nil {
		http.Error(w, "station not found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"Station": st,
		"Port":    h.port,
	}

	if err := r.ParseForm(); err != nil {
		data["Error"] = "bad form"
		h.tmpl.ExecuteTemplate(w, "sources-list", data)
		return
	}
	source := r.FormValue("source")
	if source == "" {
		data["Error"] = "source required"
		h.tmpl.ExecuteTemplate(w, "sources-list", data)
		return
	}

	if err := h.mgr.AddSource(name, source); err != nil {
		data["Error"] = err.Error()
		h.tmpl.ExecuteTemplate(w, "sources-list", data)
		return
	}

	// Refresh station after mutation
	st = h.mgr.Find(name)
	data["Station"] = st
	h.tmpl.ExecuteTemplate(w, "sources-list", data)
}

func (h *Handler) removeSource(w http.ResponseWriter, r *http.Request, name string, index int) {
	st := h.mgr.Find(name)
	if st == nil {
		http.Error(w, "station not found", http.StatusNotFound)
		return
	}
	data := map[string]any{"Station": st, "Port": h.port}

	if err := h.mgr.RemoveSource(name, index); err != nil {
		data["Error"] = err.Error()
		h.tmpl.ExecuteTemplate(w, "sources-list", data)
		return
	}

	st = h.mgr.Find(name)
	data["Station"] = st
	h.tmpl.ExecuteTemplate(w, "sources-list", data)
}
