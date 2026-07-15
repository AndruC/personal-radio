package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"radio/internal/config"
	"radio/internal/station"
)

func TestDashboardHandler(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080, MusicDir: "/tmp"},
		Stations: []config.StationConfig{
			{Name: "Rock", Mount: "/rock", Sources: []string{}},
		},
	}
	config.Save(cfgPath, cfg)

	mgr, err := station.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	h := NewHandler(mgr, 8080, "/tmp")

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Rock") {
		t.Error("response should contain station name 'Rock'")
	}
}

func TestStationHandler(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	musicDir := filepath.Join(dir, "music")
	os.MkdirAll(filepath.Join(musicDir, "Jazz"), 0755)
	os.WriteFile(filepath.Join(musicDir, "Jazz", "cool.mp3"), []byte("dummy"), 0644)

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080, MusicDir: musicDir},
		Stations: []config.StationConfig{
			{Name: "Jazz", Mount: "/jazz", Sources: []string{"Jazz"}},
		},
	}
	config.Save(cfgPath, cfg)

	mgr, err := station.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	h := NewHandler(mgr, 8080, musicDir)

	req := httptest.NewRequest("GET", "/stations/Jazz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Jazz") {
		t.Error("response should contain source 'Jazz'")
	}
}

func TestLibraryHandler(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "song.mp3"), []byte("dummy"), 0644)
	os.WriteFile(filepath.Join(dir, "tune.ogg"), []byte("dummy"), 0644)

	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{
		Server:   config.ServerConfig{Port: 8080, MusicDir: dir},
		Stations: []config.StationConfig{},
	}
	config.Save(cfgPath, cfg)

	mgr, err := station.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	h := NewHandler(mgr, 8080, dir)

	req := httptest.NewRequest("GET", "/library", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "song") {
		t.Error("response should contain track 'song'")
	}
}

func TestAddSourceHandler(t *testing.T) {
	dir := t.TempDir()
	musicDir := filepath.Join(dir, "music")
	os.MkdirAll(filepath.Join(musicDir, "NewFolder"), 0755)
	os.WriteFile(filepath.Join(musicDir, "NewFolder", "a.mp3"), []byte("dummy"), 0644)

	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080, MusicDir: musicDir},
		Stations: []config.StationConfig{
			{Name: "Test", Mount: "/test", Sources: []string{}},
		},
	}
	config.Save(cfgPath, cfg)

	mgr, err := station.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	h := NewHandler(mgr, 8080, musicDir)

	form := strings.NewReader("source=NewFolder")
	req := httptest.NewRequest("POST", "/stations/Test/sources", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestDeleteSourceHandler(t *testing.T) {
	dir := t.TempDir()
	musicDir := filepath.Join(dir, "music")
	os.MkdirAll(filepath.Join(musicDir, "folder1"), 0755)

	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080, MusicDir: musicDir},
		Stations: []config.StationConfig{
			{Name: "Test", Mount: "/test", Sources: []string{"folder1"}},
		},
	}
	config.Save(cfgPath, cfg)

	mgr, err := station.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	h := NewHandler(mgr, 8080, musicDir)

	req := httptest.NewRequest("DELETE", "/stations/Test/sources/0", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
