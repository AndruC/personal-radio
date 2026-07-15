package station

import (
	"os"
	"path/filepath"
	"testing"

	"radio/internal/config"
)

func TestManagerCreateStation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	musicDir := filepath.Join(dir, "music")
	os.Mkdir(musicDir, 0755)

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 9090, MusicDir: musicDir},
		Stations: []config.StationConfig{
			{Name: "Test", Mount: "/test", Sources: []string{"."}},
		},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	if len(mgr.AllStations()) != 1 {
		t.Fatalf("station count = %d, want 1", len(mgr.AllStations()))
	}

	st := mgr.Find("Test")
	if st == nil {
		t.Fatal("Find(Test) returned nil")
	}
	if st.Name() != "Test" {
		t.Errorf("name = %s, want Test", st.Name())
	}
	if st.Mount() != "test" {
		t.Errorf("mount = %s, want test", st.Mount())
	}
}

func TestManagerResolvesSources(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	musicDir := filepath.Join(dir, "music")
	rockDir := filepath.Join(musicDir, "Rock")
	os.MkdirAll(rockDir, 0755)

	os.WriteFile(filepath.Join(musicDir, "root.mp3"), []byte("dummy"), 0644)
	os.WriteFile(filepath.Join(rockDir, "song1.mp3"), []byte("dummy"), 0644)
	os.WriteFile(filepath.Join(rockDir, "notmusic.txt"), []byte("dummy"), 0644)

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 9090, MusicDir: musicDir},
		Stations: []config.StationConfig{
			{Name: "Rock", Mount: "/rock", Sources: []string{"Rock"}},
		},
	}
	config.Save(cfgPath, cfg)

	mgr, err := NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	st := mgr.Find("Rock")
	if st == nil {
		t.Fatal("Find(Rock) returned nil")
	}
	if st.TrackCount() != 1 {
		t.Errorf("TrackCount = %d, want 1", st.TrackCount())
	}
}

func TestManagerAddRemoveSource(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	musicDir := filepath.Join(dir, "music")
	os.MkdirAll(filepath.Join(musicDir, "Jazz"), 0755)
	os.WriteFile(filepath.Join(musicDir, "Jazz", "cool.mp3"), []byte("dummy"), 0644)

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 9090, MusicDir: musicDir},
		Stations: []config.StationConfig{
			{Name: "Test", Mount: "/test", Sources: []string{}},
		},
	}
	config.Save(cfgPath, cfg)

	mgr, _ := NewManager(cfgPath)

	if err := mgr.AddSource("Test", "Jazz"); err != nil {
		t.Fatalf("AddSource() error: %v", err)
	}

	st := mgr.Find("Test")
	if st.TrackCount() != 1 {
		t.Errorf("TrackCount = %d after add, want 1", st.TrackCount())
	}
	if len(st.Sources()) != 1 {
		t.Errorf("sources = %d, want 1", len(st.Sources()))
	}

	if err := mgr.RemoveSource("Test", 0); err != nil {
		t.Fatalf("RemoveSource() error: %v", err)
	}
	st = mgr.Find("Test")
	if st.TrackCount() != 0 {
		t.Errorf("TrackCount = %d after remove, want 0", st.TrackCount())
	}
}

func TestManagerUnknownStation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := &config.Config{
		Server:   config.ServerConfig{Port: 9090, MusicDir: "/tmp"},
		Stations: []config.StationConfig{},
	}
	config.Save(cfgPath, cfg)

	mgr, _ := NewManager(cfgPath)
	err := mgr.AddSource("NoSuch", "whatever")
	if err == nil {
		t.Error("expected error for unknown station")
	}
}
