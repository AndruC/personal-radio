package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	yaml := `
server:
  port: 9090
  music_dir: "/music"
stations:
  - name: "Rock"
    mount: "/rock"
    sources:
      - "Rock"
      - "favorites/song.mp3"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.MusicDir != "/music" {
		t.Errorf("music_dir = %s, want /music", cfg.Server.MusicDir)
	}
	if len(cfg.Stations) != 1 {
		t.Fatalf("stations count = %d, want 1", len(cfg.Stations))
	}
	if cfg.Stations[0].Name != "Rock" {
		t.Errorf("station name = %s, want Rock", cfg.Stations[0].Name)
	}
	if cfg.Stations[0].Mount != "/rock" {
		t.Errorf("mount = %s, want /rock", cfg.Stations[0].Mount)
	}
	if len(cfg.Stations[0].Sources) != 2 {
		t.Errorf("sources count = %d, want 2", len(cfg.Stations[0].Sources))
	}
}

func TestSaveConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080, MusicDir: "/music"},
		Stations: []StationConfig{
			{Name: "Test", Mount: "/test", Sources: []string{"dir1"}},
		},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after save error: %v", err)
	}
	if loaded.Server.Port != 8080 {
		t.Errorf("port = %d, want 8080", loaded.Server.Port)
	}
	if len(loaded.Stations) != 1 {
		t.Errorf("stations count = %d, want 1", len(loaded.Stations))
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
