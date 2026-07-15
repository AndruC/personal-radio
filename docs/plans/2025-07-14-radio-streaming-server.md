# Radio Streaming Server — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go server that streams random MP3/OGG music from local folders as SHOUTcast-compatible radio stations, managed via an htmx web UI.

**Architecture:** A config-driven server with per-station playlist engines feeding embedded SHOUTcast streamers. A minimal htmx UI manages station sources. Single binary, no external dependencies beyond the Go toolchain.

**Tech Stack:** Go 1.22+, `gopkg.in/yaml.v3` for config, `github.com/dhowden/tag` for ID3/Vorbis tags, `embed` for HTML templates, standard library `net/http`.

---

## File Structure

```
radio/
├── main.go                       # Entry point: parse flags, init config, start server
├── go.mod
├── go.sum
├── config.yaml                   # Default config
├── internal/
│   ├── config/
│   │   ├── config.go             # Types + YAML load/save
│   │   └── config_test.go
│   ├── scanner/
│   │   ├── scanner.go            # Walk music dir, index MP3/OGG files
│   │   └── scanner_test.go
│   ├── playlist/
│   │   ├── playlist.go           # Shuffled track queue with next/loop
│   │   └── playlist_test.go
│   ├── streamer/
│   │   ├── streamer.go           # SHOUTcast ICY HTTP handler
│   │   └── streamer_test.go
│   ├── station/
│   │   ├── manager.go            # Station lifecycle: create, start, stop, update sources
│   │   └── manager_test.go
│   └── web/
│       ├── handlers.go           # htmx route handlers
│       ├── handlers_test.go
│       └── templates/
│           ├── base.html          # Base layout with htmx CDN
│           ├── dashboard.html     # Station list + now-playing
│           ├── station.html       # Station detail: sources and tracks
│           └── library.html       # Library browser with search
```

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go` (stub)
- Create: `internal/` directory tree
- Create: `config.yaml`

- [ ] **Step 1: Initialize Go module**

```bash
cd C:/Users/Andrew/Projects/radio
go mod init radio
```

Expected: `go.mod` created with `module radio`

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p internal/config internal/scanner internal/playlist internal/streamer internal/station internal/web/templates
```

- [ ] **Step 3: Create stub main.go**

Create `main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("radio server starting...")
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build -o radio.exe .
```

Expected: compilation succeeds, `radio.exe` created.

- [ ] **Step 5: Create default config.yaml**

Create `config.yaml`:
```yaml
server:
  port: 8080
  music_dir: "P:\\My Media\\My Music"

stations:
  - name: "Everything"
    mount: "/everything"
    sources:
      - "."
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "chore: project scaffolding"
```

---

### Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for config loading**

Create `internal/config/config_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/config/ -v
```

Expected: compilation error — `Config`, `Load`, `Save` not defined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/config/config.go`:
```go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port     int    `yaml:"port"`
	MusicDir string `yaml:"music_dir"`
}

type StationConfig struct {
	Name    string   `yaml:"name"`
	Mount   string   `yaml:"mount"`
	Sources []string `yaml:"sources"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Stations []StationConfig `yaml:"stations"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 4: Install yaml dependency and run tests**

```bash
go get gopkg.in/yaml.v3
go test ./internal/config/ -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: config package with YAML load/save"
```

---

### Task 3: Scanner Package

**Files:**
- Create: `internal/scanner/scanner.go`
- Create: `internal/scanner/scanner_test.go`

- [ ] **Step 1: Write failing test for scanner**

Create `internal/scanner/scanner_test.go`:
```go
package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func createTestFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanFindsMP3AndOGG(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "song1.mp3")
	createTestFile(t, dir, "song2.MP3")
	createTestFile(t, dir, "song3.ogg")
	createTestFile(t, dir, "song4.OGG")
	createTestFile(t, dir, "notmusic.txt")
	createTestFile(t, dir, "other.flac")

	results, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("found %d files, want 4", len(results))
	}
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = filepath.Base(r.Path)
	}
	sort.Strings(names)
	expected := []string{"song1.mp3", "song2.MP3", "song3.ogg", "song4.OGG"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("names[%d] = %s, want %s", i, names[i], want)
		}
	}
}

func TestScanRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	os.Mkdir(sub, 0755)
	createTestFile(t, dir, "root.mp3")
	createTestFile(t, sub, "nested.ogg")

	results, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("found %d files, want 2", len(results))
	}
}

func TestScanEmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("found %d files, want 0", len(results))
	}
}

func TestScanMissingDir(t *testing.T) {
	_, err := Scan("/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing directory")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/scanner/ -v
```

Expected: compilation error — package or types not defined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/scanner/scanner.go`:
```go
package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

type Track struct {
	Path string // absolute path to the file
	Name string // display name (filename without extension)
}

var supportedExts = map[string]bool{
	".mp3": true,
	".ogg": true,
}

func Scan(root string) ([]Track, error) {
	var tracks []Track
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors for individual files
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if supportedExts[ext] {
			tracks = append(tracks, Track{
				Path: path,
				Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tracks, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/scanner/ -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/
git commit -m "feat: music scanner for MP3/OGG files"
```

---

### Task 4: Playlist Engine

**Files:**
- Create: `internal/playlist/playlist.go`
- Create: `internal/playlist/playlist_test.go`

- [ ] **Step 1: Write failing test for playlist**

Create `internal/playlist/playlist_test.go`:
```go
package playlist

import (
	"slices"
	"testing"
)

func TestNewPlaylist(t *testing.T) {
	tracks := []string{
		"/music/a.mp3",
		"/music/b.ogg",
		"/music/c.mp3",
	}
	p := New(tracks)
	if p.Len() != 3 {
		t.Errorf("Len() = %d, want 3", p.Len())
	}
}

func TestPlaylistNextLoops(t *testing.T) {
	tracks := []string{"/music/a.mp3", "/music/b.ogg"}
	p := New(tracks)

	seen := make(map[string]bool)
	for i := 0; i < 4; i++ {
		track, ok := p.Next()
		if !ok {
			t.Fatal("Next() returned false on non-empty playlist")
		}
		seen[track] = true
	}
	if len(seen) != 2 {
		t.Errorf("did not see all tracks: saw %v", seen)
	}
}

func TestPlaylistShuffles(t *testing.T) {
	// Create a playlist with many tracks and verify order differs from input order
	tracks := make([]string, 100)
	for i := range tracks {
		tracks[i] = string(rune('A' + i%26)) + string(rune('a'+i/26))
	}

	p := New(tracks)
	first := make([]string, 5)
	for i := range first {
		track, ok := p.Next()
		if !ok {
			t.Fatal("Next() returned false")
		}
		first[i] = track
	}

	// It's possible (but astronomically unlikely) that the shuffle preserves order
	// for 5 elements. We just check that it's not exactly the first 5 in order.
	if slices.Equal(first, tracks[:5]) {
		// Very unlikely — try one more shuffle
		p2 := New(tracks)
		var second []string
		for i := 0; i < 5; i++ {
			t, _ := p2.Next()
			second = append(second, t)
		}
		if slices.Equal(second, tracks[:5]) {
			t.Skip("shuffle preserved order twice — extremely unlikely, skipping")
		}
	}
}

func TestPlaylistEmpty(t *testing.T) {
	p := New(nil)
	_, ok := p.Next()
	if ok {
		t.Error("Next() on empty playlist should return false")
	}
}

func TestPlaylistCurrent(t *testing.T) {
	p := New([]string{"/music/a.mp3", "/music/b.ogg"})
	track, _ := p.Next()
	if p.Current() != track {
		t.Errorf("Current() = %s, want %s", p.Current(), track)
	}
	// Current should not advance
	if p.Current() != track {
		t.Errorf("Current() changed after second call")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/playlist/ -v
```

Expected: compilation error.

- [ ] **Step 3: Write minimal implementation**

Create `internal/playlist/playlist.go`:
```go
package playlist

import "math/rand"

type Playlist struct {
	tracks  []string
	current int
}

func New(tracks []string) *Playlist {
	shuffled := make([]string, len(tracks))
	copy(shuffled, tracks)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return &Playlist{
		tracks:  shuffled,
		current: -1,
	}
}

func (p *Playlist) Next() (string, bool) {
	if len(p.tracks) == 0 {
		return "", false
	}
	p.current = (p.current + 1) % len(p.tracks)
	return p.tracks[p.current], true
}

func (p *Playlist) Current() string {
	if p.current < 0 || len(p.tracks) == 0 {
		return ""
	}
	return p.tracks[p.current]
}

func (p *Playlist) Len() int {
	return len(p.tracks)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/playlist/ -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/playlist/
git commit -m "feat: playlist engine with shuffle and loop"
```

---

### Task 5: SHOUTcast Streamer

**Files:**
- Create: `internal/streamer/streamer.go`
- Create: `internal/streamer/streamer_test.go`

- [ ] **Step 1: Write failing test for streamer**

Create `internal/streamer/streamer_test.go`:
```go
package streamer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createSilentMP3 creates a minimal valid MP3 file (frame header + silence)
func createSilentMP3(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	// Minimal MPEG1 Layer 3 frame header (128kbps, 44100Hz, stereo) + 417 bytes of silence
	frame := []byte{
		0xFF, 0xFB, 0x90, 0x00, // MPEG1, Layer3, 128kbps, 44100, stereo
	}
	frame = append(frame, make([]byte, 413)...) // padding to 417 byte frame
	if err := os.WriteFile(path, frame, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

type fakePlaylist struct {
	tracks []string
	pos    int
}

func (f *fakePlaylist) Next() (string, bool) {
	if f.pos >= len(f.tracks) {
		return "", false
	}
	track := f.tracks[f.pos]
	f.pos++
	return track, true
}

func (f *fakePlaylist) Current() string {
	if f.pos > 0 && f.pos <= len(f.tracks) {
		return f.tracks[f.pos-1]
	}
	return ""
}

func TestStreamerICYHeaders(t *testing.T) {
	dir := t.TempDir()
	path := createSilentMP3(t, dir, "test.mp3")

	fp := &fakePlaylist{tracks: []string{path}}
	srv := New(fp, "Test Station")

	req := httptest.NewRequest("GET", "/stream/test", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	resp := rec.Result()
	contentType := resp.Header.Get("Content-Type")
	if contentType != "audio/mpeg" {
		t.Errorf("Content-Type = %s, want audio/mpeg", contentType)
	}
	icyName := resp.Header.Get("icy-name")
	if icyName != "Test Station" {
		t.Errorf("icy-name = %s, want Test Station", icyName)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestStreamerSendsAudioData(t *testing.T) {
	dir := t.TempDir()
	path := createSilentMP3(t, dir, "song.mp3")
	fileData, _ := os.ReadFile(path)

	fp := &fakePlaylist{tracks: []string{path}}
	srv := New(fp, "Radio")

	req := httptest.NewRequest("GET", "/stream/radio", nil)
	rec := httptest.NewRecorder()

	// Stream in a goroutine since ServeHTTP blocks
	done := make(chan struct{})
	go func() {
		srv.ServeHTTP(rec, req)
		close(done)
	}()

	// Let it stream a bit then check
	// The streamer will loop the single track; read a chunk
	<-done // wait for it to finish (it won't unless context cancels, but for test we check partial)

	body := rec.Body.Bytes()
	if len(body) == 0 {
		t.Error("expected audio data, got empty body")
	}
	_ = fileData
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/streamer/ -v
```

Expected: compilation error.

- [ ] **Step 3: Write minimal implementation**

Create `internal/streamer/streamer.go`:
```go
package streamer

import (
	"io"
	"log"
	"net/http"
	"os"
)

type PlaylistProvider interface {
	Next() (string, bool)
	Current() string
}

type Streamer struct {
	playlist PlaylistProvider
	name     string
}

func New(playlist PlaylistProvider, name string) *Streamer {
	return &Streamer{
		playlist: playlist,
		name:     name,
	}
}

func (s *Streamer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("icy-name", s.name)
	w.Header().Set("icy-pub", "0")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	for {
		trackPath, ok := s.playlist.Next()
		if !ok {
			// No tracks, send silence or just break
			return
		}

		err := s.streamFile(w, trackPath, canFlush, flusher)
		if err != nil {
			log.Printf("stream error on %s: %v", trackPath, err)
			return
		}
	}
}

func (s *Streamer) streamFile(w io.Writer, path string, canFlush bool, flusher http.Flusher) error {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("cannot open %s: %v", path, err)
		return nil // skip unreadable files
	}
	defer f.Close()

	buf := make([]byte, 16*1024) // 16KB chunks
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr // client disconnected
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/streamer/ -v -timeout 10s
```

The streamer goroutine test will time out because ServeHTTP blocks forever by design. Adjust the test:

- [ ] **Step 4a: Fix the streaming test to be more practical**

Replace `TestStreamerSendsAudioData` in `internal/streamer/streamer_test.go`:
```go
func TestStreamerSendsAudioData(t *testing.T) {
	dir := t.TempDir()
	path := createSilentMP3(t, dir, "song.mp3")

	fp := &fakePlaylist{tracks: []string{path}}
	srv := New(fp, "Radio")

	// Use a pipe so we can read a bit then cancel
	pr, pw := io.Pipe()
	req := httptest.NewRequest("GET", "/stream/radio", nil)

	go func() {
		srv.ServeHTTP(
			&responseWriterShim{w: pw, header: make(http.Header)},
			req,
		)
	}()

	buf := make([]byte, 1024)
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if n == 0 {
		t.Error("expected audio data, got 0 bytes")
	}
	pr.Close()
}

// responseWriterShim adapts io.WriteCloser to http.ResponseWriter just enough for the streamer
type responseWriterShim struct {
	w      io.WriteCloser
	header http.Header
	code   int
}

func (s *responseWriterShim) Header() http.Header         { return s.header }
func (s *responseWriterShim) Write(b []byte) (int, error)  { return s.w.Write(b) }
func (s *responseWriterShim) WriteHeader(code int)         { s.code = code }
```

- [ ] **Step 4b: Run tests again**

```bash
go test ./internal/streamer/ -v -timeout 10s
```

Expected: tests pass (ICY headers test passes, audio data test reads bytes then closes).

- [ ] **Step 5: Commit**

```bash
git add internal/streamer/
git commit -m "feat: SHOUTcast/ICY streamer"
```

---

### Task 6: Station Manager

**Files:**
- Create: `internal/station/manager.go`
- Create: `internal/station/manager_test.go`
- Modify: `internal/streamer/streamer.go` (add GetName accessor)

- [ ] **Step 1: Add GetName to Streamer**

In `internal/streamer/streamer.go`, add after the `New` function:
```go
func (s *Streamer) Name() string {
	return s.name
}
```

- [ ] **Step 2: Write failing test for station manager**

Create `internal/station/manager_test.go`:
```go
package station

import (
	"os"
	"path/filepath"
	"testing"

	"radio/internal/config"
	"radio/internal/scanner"
)

func TestManagerCreateStation(t *testing.T) {
	dir := t.TempDir()
	// Create config
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

	if len(mgr.Stations()) != 1 {
		t.Fatalf("station count = %d, want 1", len(mgr.Stations()))
	}

	st := mgr.Stations()[0]
	if st.Name != "Test" {
		t.Errorf("name = %s, want Test", st.Name)
	}
	if st.Mount != "/test" {
		t.Errorf("mount = %s, want /test", st.Mount)
	}
}

func TestManagerResolvesSources(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	musicDir := filepath.Join(dir, "music")
	rockDir := filepath.Join(musicDir, "Rock")
	os.MkdirAll(rockDir, 0755)

	// Create some test MP3 files
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

	// Check that the station resolved its sources
	st := mgr.Stations()[0]
	if st.TrackCount != 1 {
		t.Errorf("TrackCount = %d, want 1 (only Rock/song1.mp3, root.mp3 not in Rock/)", st.TrackCount)
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

	mgr, err := NewManager(cfgPath)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Add a source
	if err := mgr.AddSource("Test", "Jazz"); err != nil {
		t.Fatalf("AddSource() error: %v", err)
	}

	st := mgr.Stations()[0]
	if st.TrackCount != 1 {
		t.Errorf("TrackCount = %d after add, want 1", st.TrackCount)
	}
	if len(st.Config.Sources) != 1 {
		t.Errorf("sources = %d, want 1", len(st.Config.Sources))
	}

	// Remove the source
	if err := mgr.RemoveSource("Test", 0); err != nil {
		t.Fatalf("RemoveSource() error: %v", err)
	}
	st = mgr.Stations()[0]
	if st.TrackCount != 0 {
		t.Errorf("TrackCount = %d after remove, want 0", st.TrackCount)
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

// Avoid unused import for scanner
var _ = scanner.Scan
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/station/ -v
```

Expected: compilation error — `NewManager`, `StationInfo`, etc. not defined.

- [ ] **Step 4: Write minimal implementation**

Create `internal/station/manager.go`:
```go
package station

import (
	"fmt"
	"log"
	"path/filepath"

	"radio/internal/config"
	"radio/internal/playlist"
	"radio/internal/scanner"
	"radio/internal/streamer"
)

type StationInfo struct {
	Name       string
	Mount      string
	TrackCount int
	Config     config.StationConfig
	streamer   *streamer.Streamer
	playlist   *playlist.Playlist
}

type Manager struct {
	configPath string
	cfg        *config.Config
	stations   []*StationInfo
}

func NewManager(configPath string) (*Manager, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	m := &Manager{
		configPath: configPath,
		cfg:        cfg,
	}

	for _, sc := range cfg.Stations {
		if _, err := m.createStation(sc); err != nil {
			log.Printf("warning: station %s: %v", sc.Name, err)
		}
	}

	return m, nil
}

func (m *Manager) createStation(sc config.StationConfig) (*StationInfo, error) {
	tracks, err := m.resolveSources(sc.Sources)
	if err != nil {
		return nil, fmt.Errorf("resolve sources: %w", err)
	}

	pl := playlist.New(tracks)
	s := streamer.New(pl, sc.Name)

	si := &StationInfo{
		Name:       sc.Name,
		Mount:      sc.Mount,
		TrackCount: len(tracks),
		Config:     sc,
		streamer:   s,
		playlist:   pl,
	}

	m.stations = append(m.stations, si)
	return si, nil
}

func (m *Manager) resolveSources(sources []string) ([]string, error) {
	var allTracks []string
	for _, src := range sources {
		fullPath := src
		if !filepath.IsAbs(src) {
			fullPath = filepath.Join(m.cfg.Server.MusicDir, src)
		}
		tracks, err := scanner.Scan(fullPath)
		if err != nil {
			log.Printf("warning: scanning %s: %v", src, err)
			continue
		}
		for _, t := range tracks {
			allTracks = append(allTracks, t.Path)
		}
	}
	return allTracks, nil
}

func (m *Manager) Stations() []*StationInfo {
	return m.stations
}

func (m *Manager) FindStation(name string) *StationInfo {
	for _, s := range m.stations {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func (m *Manager) AddSource(stationName, source string) error {
	st := m.FindStation(stationName)
	if st == nil {
		return fmt.Errorf("station %q not found", stationName)
	}

	// Resolve the new source
	fullPath := source
	if !filepath.IsAbs(source) {
		fullPath = filepath.Join(m.cfg.Server.MusicDir, source)
	}
	newTracks, err := scanner.Scan(fullPath)
	if err != nil {
		return fmt.Errorf("scan source: %w", err)
	}

	// Update config
	st.Config.Sources = append(st.Config.Sources, source)
	st.TrackCount += len(newTracks)

	// Rebuild playlist with all tracks
	allTracks, _ := m.resolveSources(st.Config.Sources)
	st.playlist = playlist.New(allTracks)
	st.streamer = streamer.New(st.playlist, st.Name)

	// Save config
	return m.saveConfig()
}

func (m *Manager) RemoveSource(stationName string, index int) error {
	st := m.FindStation(stationName)
	if st == nil {
		return fmt.Errorf("station %q not found", stationName)
	}
	if index < 0 || index >= len(st.Config.Sources) {
		return fmt.Errorf("source index %d out of range", index)
	}

	st.Config.Sources = append(st.Config.Sources[:index], st.Config.Sources[index+1:]...)

	// Rebuild playlist
	allTracks, _ := m.resolveSources(st.Config.Sources)
	st.TrackCount = len(allTracks)
	st.playlist = playlist.New(allTracks)
	st.streamer = streamer.New(st.playlist, st.Name)

	return m.saveConfig()
}

func (m *Manager) StreamerFor(stationName string) *streamer.Streamer {
	st := m.FindStation(stationName)
	if st == nil {
		return nil
	}
	return st.streamer
}

func (m *Manager) saveConfig() error {
	for _, st := range m.stations {
		for i, sc := range m.cfg.Stations {
			if sc.Name == st.Name {
				m.cfg.Stations[i].Sources = st.Config.Sources
				break
			}
		}
	}
	return config.Save(m.configPath, m.cfg)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/station/ -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/station/ internal/streamer/streamer.go
git commit -m "feat: station manager with source add/remove"
```

---

### Task 7: Web UI Handlers

**Files:**
- Create: `internal/web/templates/base.html`
- Create: `internal/web/templates/dashboard.html`
- Create: `internal/web/templates/station.html`
- Create: `internal/web/templates/library.html`
- Create: `internal/web/handlers.go`
- Create: `internal/web/handlers_test.go`

- [ ] **Step 1: Write HTML templates**

Create `internal/web/templates/base.html`:
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Radio Server</title>
    <script src="https://unpkg.com/htmx.org@1.9.12"></script>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 1rem; }
        table { width: 100%; border-collapse: collapse; }
        th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid #ddd; }
        .now-playing { color: #666; font-style: italic; }
        button, .btn { padding: 0.3rem 0.6rem; cursor: pointer; }
        .flash { padding: 0.5rem; margin: 0.5rem 0; border-radius: 4px; }
        .flash-success { background: #d4edda; }
        .flash-error { background: #f8d7da; }
        input[type="search"] { padding: 0.3rem; width: 100%; margin-bottom: 0.5rem; }
    </style>
</head>
<body>
    <nav>
        <a href="/">Dashboard</a> |
        <a href="/library">Library</a>
    </nav>
    <hr>
    {{block "content" .}}{{end}}
</body>
</html>
```

Create `internal/web/templates/dashboard.html`:
```html
{{define "content"}}
<h1>Stations</h1>
{{range .Stations}}
<div style="margin-bottom: 1rem; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;">
    <h2 style="margin: 0;">
        <a href="/stations/{{.Name}}">{{.Name}}</a>
    </h2>
    <p class="now-playing" hx-get="/stations/{{.Name}}/status" hx-trigger="every 5s">
        {{.TrackCount}} tracks
    </p>
    <p>Stream URL: <code>http://localhost:{{$.Port}}/stream/{{.Mount}}</code></p>
</div>
{{else}}
<p>No stations configured. Edit <code>config.yaml</code> to add stations.</p>
{{end}}
{{end}}
```

Create `internal/web/templates/station.html`:
```html
{{define "content"}}
<h1>{{.Station.Name}}</h1>
<p>Mount: <code>{{.Station.Mount}}</code> | Tracks: {{.Station.TrackCount}}</p>
<p>Stream URL: <code>http://localhost:{{.Port}}/stream/{{.Station.Mount}}</code></p>

<h2>Sources</h2>
<ul id="sources-list">
{{range $i, $src := .Station.Config.Sources}}
    <li>
        {{$src}}
        <button hx-delete="/stations/{{$.Station.Name}}/sources/{{$i}}"
                hx-target="#sources-list"
                hx-swap="outerHTML">Remove</button>
    </li>
{{else}}
    <li>No sources. Add a folder or file below.</li>
{{end}}
</ul>

<h3>Add Source</h3>
<form hx-post="/stations/{{.Station.Name}}/sources"
      hx-target="#sources-list"
      hx-swap="outerHTML">
    <input type="text" name="source" placeholder="Folder or file path" required>
    <button type="submit">Add</button>
</form>

<p><a href="/">← Back to Dashboard</a></p>
{{end}}
```

Create `internal/web/templates/library.html`:
```html
{{define "content"}}
<h1>Music Library</h1>

<input type="search" name="q"
       hx-get="/library"
       hx-trigger="keyup changed delay:200ms"
       hx-target="#results"
       hx-swap="innerHTML"
       placeholder="Search tracks...">

<div id="results">
{{template "library-results" .}}
</div>
{{end}}

{{define "library-results"}}
<p>{{len .Tracks}} tracks found{{if .Query}} for "{{.Query}}"{{end}}</p>
<table>
    <thead><tr><th>Name</th><th>Path</th><th>Add to Station</th></tr></thead>
    <tbody>
    {{range .Tracks}}
    <tr>
        <td>{{.Name}}</td>
        <td style="font-size: 0.85em; color: #666;">{{.Path}}</td>
        <td>
            <form hx-post="/stations/add-track" hx-target="this" hx-swap="outerHTML" style="display:inline;">
                <input type="hidden" name="track" value="{{.Path}}">
                <select name="station">
                    {{range $.Stations}}
                    <option value="{{.Name}}">{{.Name}}</option>
                    {{end}}
                </select>
                <button type="submit">Add</button>
            </form>
        </td>
    </tr>
    {{else}}
    <tr><td colspan="3">No tracks found.</td></tr>
    {{end}}
    </tbody>
</table>
{{end}}
```

- [ ] **Step 2: Write failing test for handlers**

Create `internal/web/handlers_test.go`:
```go
package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"radio/internal/config"
	"radio/internal/scanner"
)

type testManager struct {
	stations []*testStation
}

type testStation struct {
	name       string
	mount      string
	trackCount int
	sources    []string
	tracks     []scanner.Track
}

func (m *testManager) Stations() []StationProvider {
	result := make([]StationProvider, len(m.stations))
	for i, s := range m.stations {
		result[i] = s
	}
	return result
}

func (m *testManager) FindStation(name string) StationProvider {
	for _, s := range m.stations {
		if s.name == name {
			return s
		}
	}
	return nil
}

func (m *testManager) AddSource(stationName, source string) error {
	for _, s := range m.stations {
		if s.name == stationName {
			s.sources = append(s.sources, source)
			return nil
		}
	}
	return nil
}

func (m *testManager) RemoveSource(stationName string, index int) error {
	for _, s := range m.stations {
		if s.name == stationName {
			if index >= 0 && index < len(s.sources) {
				s.sources = append(s.sources[:index], s.sources[index+1:]...)
			}
			return nil
		}
	}
	return nil
}

func (m *testManager) AddTrack(stationName, trackPath string) error {
	return nil
}

func (s *testStation) Name() string             { return s.name }
func (s *testStation) Mount() string            { return s.mount }
func (s *testStation) TrackCount() int          { return s.trackCount }
func (s *testStation) Sources() []string        { return s.sources }
func (s *testStation) ConfigSources() []string  { return s.sources }

func TestDashboardHandler(t *testing.T) {
	mgr := &testManager{
		stations: []*testStation{
			{name: "Rock", mount: "/rock", trackCount: 42},
		},
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
	if !strings.Contains(body, "42 tracks") {
		t.Error("response should contain track count")
	}
}

func TestStationHandler(t *testing.T) {
	mgr := &testManager{
		stations: []*testStation{
			{name: "Jazz", mount: "/jazz", trackCount: 7, sources: []string{"Jazz", "More Jazz"}},
		},
	}
	h := NewHandler(mgr, 8080, "/tmp")

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
	if !strings.Contains(body, "More Jazz") {
		t.Error("response should contain source 'More Jazz'")
	}
}

func TestLibraryHandler(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "song.mp3"), []byte("dummy"), 0644)
	os.WriteFile(filepath.Join(dir, "tune.ogg"), []byte("dummy"), 0644)

	mgr := &testManager{
		stations: []*testStation{
			{name: "Rock", mount: "/rock", trackCount: 0},
		},
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
	mgr := &testManager{
		stations: []*testStation{
			{name: "Test", mount: "/test", trackCount: 0, sources: []string{}},
		},
	}
	h := NewHandler(mgr, 8080, "/tmp")

	form := strings.NewReader("source=MyFolder")
	req := httptest.NewRequest("POST", "/stations/Test/sources", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestDeleteSourceHandler(t *testing.T) {
	mgr := &testManager{
		stations: []*testStation{
			{name: "Test", mount: "/test", trackCount: 0, sources: []string{"folder1"}},
		},
	}
	h := NewHandler(mgr, 8080, "/tmp")

	req := httptest.NewRequest("DELETE", "/stations/Test/sources/0", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/web/ -v
```

Expected: compilation error — `NewHandler`, `StationProvider`, etc. not defined.

- [ ] **Step 4: Write implementation**

Create `internal/web/handlers.go`:
```go
package web

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"radio/internal/scanner"
)

//go:embed templates/*.html
var templateFS embed.FS

type StationProvider interface {
	Name() string
	Mount() string
	TrackCount() int
	Sources() []string
	ConfigSources() []string
}

type Manager interface {
	Stations() []StationProvider
	FindStation(name string) StationProvider
	AddSource(stationName, source string) error
	RemoveSource(stationName string, index int) error
}

type Handler struct {
	mgr      Manager
	port     int
	musicDir string
	tmpl     *template.Template
}

func NewHandler(mgr Manager, port int, musicDir string) *Handler {
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
	case path == "/library" && r.Method == "GET":
		h.library(w, r)
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
		"Stations": h.mgr.Stations(),
		"Port":     h.port,
	}
	h.tmpl.ExecuteTemplate(w, "base.html", nil)
	h.tmpl.ExecuteTemplate(w, "dashboard.html", data)
}

func (h *Handler) stationDetail(w http.ResponseWriter, r *http.Request, name string) {
	st := h.mgr.FindStation(name)
	if st == nil {
		http.NotFound(w, r)
		return
	}
	data := map[string]any{
		"Station": st,
		"Port":    h.port,
	}
	h.tmpl.ExecuteTemplate(w, "base.html", nil)
	h.tmpl.ExecuteTemplate(w, "station.html", data)
}

func (h *Handler) stationStatus(w http.ResponseWriter, r *http.Request, name string) {
	st := h.mgr.FindStation(name)
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
		"Stations": h.mgr.Stations(),
	}

	if query != "" {
		// htmx partial render
		h.tmpl.ExecuteTemplate(w, "library-results", data)
		return
	}

	h.tmpl.ExecuteTemplate(w, "base.html", nil)
	h.tmpl.ExecuteTemplate(w, "library.html", data)
}

func (h *Handler) addSource(w http.ResponseWriter, r *http.Request, name string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	source := r.FormValue("source")
	if source == "" {
		http.Error(w, "source required", http.StatusBadRequest)
		return
	}

	if err := h.mgr.AddSource(name, source); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated source list
	st := h.mgr.FindStation(name)
	if st == nil {
		http.Error(w, "station not found", http.StatusNotFound)
		return
	}
	data := map[string]any{
		"Station": st,
		"Port":    h.port,
	}
	h.tmpl.ExecuteTemplate(w, "station.html", data)
}

func (h *Handler) removeSource(w http.ResponseWriter, r *http.Request, name string, index int) {
	if err := h.mgr.RemoveSource(name, index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated source list
	st := h.mgr.FindStation(name)
	if st == nil {
		http.Error(w, "station not found", http.StatusNotFound)
		return
	}
	data := map[string]any{
		"Station": st,
		"Port":    h.port,
	}
	h.tmpl.ExecuteTemplate(w, "station.html", data)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/web/ -v
```

Expected: tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/web/
git commit -m "feat: htmx web UI handlers and templates"
```

---

### Task 8: Main Entry Point — Wire Everything Together

**Files:**
- Create: `main.go` (overwrite stub)
- Modify: `internal/station/manager.go` (add interface compliance)
- Create: `internal/station/interface.go` (extract interface)

- [ ] **Step 1: Extract the Manager interface for web package**

Create `internal/station/interface.go`:
```go
package station

import "radio/internal/scanner"

type StationProvider interface {
	Name() string
	Mount() string
	TrackCount() int
	Sources() []string
	ConfigSources() []string
}

// Ensure StationInfo satisfies StationProvider
var _ StationProvider = (*StationInfo)(nil)

func (s *StationInfo) Sources() []string {
	return s.Config.Sources
}

func (s *StationInfo) ConfigSources() []string {
	return s.Config.Sources
}
```

- [ ] **Step 2: Update Manager to match web.Manager interface**

Add to `internal/station/manager.go` — modify `Stations()` return type and add a cast method:

```go
func (m *Manager) Stations() []StationProvider {
	result := make([]StationProvider, len(m.stations))
	for i, s := range m.stations {
		result[i] = s
	}
	return result
}

func (m *Manager) FindStation(name string) StationProvider {
	for _, s := range m.stations {
		if s.Name == name {
			return s
		}
	}
	return nil
}
```

The existing `FindStation` returns `*StationInfo` — we need to change its return type. Update the existing method signature in `manager.go`:

Change:
```go
func (m *Manager) FindStation(name string) *StationInfo {
```
To:
```go
func (m *Manager) FindStation(name string) StationProvider {
```

And add an internal helper:
```go
func (m *Manager) findStationInfo(name string) *StationInfo {
	for _, s := range m.stations {
		if s.Name == name {
			return s
		}
	}
	return nil
}
```

Update `AddSource` and `RemoveSource` to use `findStationInfo` instead of `FindStation`.

- [ ] **Step 3: Write main.go**

Write `main.go`:
```go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"radio/internal/station"
	"radio/internal/web"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	mgr, err := station.NewManager(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	port := mgr.Port()
	handler := web.NewHandler(mgr, port, mgr.MusicDir())

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	// Register stream routes for each station
	for _, st := range mgr.AllStations() {
		streamer := mgr.StreamerFor(st.Name)
		if streamer != nil {
			mux.Handle("/stream"+st.Mount, streamer)
		}
	}

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Radio server starting on http://localhost%s", addr)
	log.Printf("Web UI: http://localhost%s", addr)
	for _, st := range mgr.AllStations() {
		log.Printf("Stream %s: http://localhost%s/stream%s", st.Name, addr, st.Mount)
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 4: Add Port() and MusicDir() and AllStations() to Manager**

Add to `internal/station/manager.go`:
```go
func (m *Manager) Port() int {
	return m.cfg.Server.Port
}

func (m *Manager) MusicDir() string {
	return m.cfg.Server.MusicDir
}

func (m *Manager) AllStations() []*StationInfo {
	return m.stations
}
```

- [ ] **Step 5: Fix web handler to not require Manager interface for now — use concrete if needed**

Actually, since `web.Manager` expects `Stations() []StationProvider` and `FindStation(name string) StationProvider`, we need to make sure the manager.go changes align.

Let's simplify — have `web.NewHandler` accept a concrete `*station.Manager`:

In `internal/web/handlers.go`, change:
```go
type Manager interface {
	Stations() []StationProvider
	FindStation(name string) StationProvider
	AddSource(stationName, source string) error
	RemoveSource(stationName string, index int) error
}

func NewHandler(mgr Manager, port int, musicDir string) *Handler {
```

To:
```go
type Manager interface {
	Stations() []StationProvider
	FindStation(name string) StationProvider
	AddSource(stationName, source string) error
	RemoveSource(stationName string, index int) error
}
```

And in `main.go`, the `station.Manager` will satisfy this interface once we fix the method signatures.

- [ ] **Step 6: Reconcile all types and compile**

The key issue is the `web.Manager` interface vs `station.Manager` concrete type. Since this is getting tangled with repeated edits, let me write the final versions of the affected files.

We need to update `internal/station/manager.go` to have these public methods that match the web.Manager interface. The `Stations()` method return type is the trickiest.

Actually, let's take a cleaner approach. Have `station.Manager` expose methods that the web package needs directly. Drop the interface in web and use a concrete reference, or define the interface in station and reference it in web.

Simplest path: define the interface in `station` package and have web use it.

Rewrite `internal/station/manager.go` with the final version:

```go
package station

import (
	"fmt"
	"log"
	"path/filepath"

	"radio/internal/config"
	"radio/internal/playlist"
	"radio/internal/scanner"
	"radio/internal/streamer"
)

type StationProvider interface {
	Name() string
	Mount() string
	TrackCount() int
	Sources() []string
}

type StationInfo struct {
	Name       string
	Mount      string
	TrackCount int
	Config     config.StationConfig
	streamer   *streamer.Streamer
	playlist   *playlist.Playlist
}

// Ensure StationInfo satisfies StationProvider
var _ StationProvider = (*StationInfo)(nil)

func (s *StationInfo) Sources() []string {
	return s.Config.Sources
}

type Manager struct {
	configPath string
	cfg        *config.Config
	stations   []*StationInfo
}

func NewManager(configPath string) (*Manager, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	m := &Manager{
		configPath: configPath,
		cfg:        cfg,
	}

	for _, sc := range cfg.Stations {
		if _, err := m.createStation(sc); err != nil {
			log.Printf("warning: station %s: %v", sc.Name, err)
		}
	}

	return m, nil
}

func (m *Manager) Port() int {
	return m.cfg.Server.Port
}

func (m *Manager) MusicDir() string {
	return m.cfg.Server.MusicDir
}

func (m *Manager) AllStations() []*StationInfo {
	return m.stations
}

func (m *Manager) StationList() []StationProvider {
	result := make([]StationProvider, len(m.stations))
	for i, s := range m.stations {
		result[i] = s
	}
	return result
}

func (m *Manager) Find(name string) StationProvider {
	s := m.find(name)
	if s == nil {
		return nil
	}
	return s
}

func (m *Manager) find(name string) *StationInfo {
	for _, s := range m.stations {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func (m *Manager) StreamerFor(name string) *streamer.Streamer {
	s := m.find(name)
	if s == nil {
		return nil
	}
	return s.streamer
}

func (m *Manager) createStation(sc config.StationConfig) (*StationInfo, error) {
	tracks, err := m.resolveSources(sc.Sources)
	if err != nil {
		return nil, fmt.Errorf("resolve sources: %w", err)
	}

	pl := playlist.New(tracks)
	s := streamer.New(pl, sc.Name)

	si := &StationInfo{
		Name:       sc.Name,
		Mount:      sc.Mount,
		TrackCount: len(tracks),
		Config:     sc,
		streamer:   s,
		playlist:   pl,
	}

	m.stations = append(m.stations, si)
	return si, nil
}

func (m *Manager) resolveSources(sources []string) ([]string, error) {
	var allTracks []string
	for _, src := range sources {
		fullPath := src
		if !filepath.IsAbs(src) {
			fullPath = filepath.Join(m.cfg.Server.MusicDir, src)
		}
		tracks, err := scanner.Scan(fullPath)
		if err != nil {
			log.Printf("warning: scanning %s: %v", src, err)
			continue
		}
		for _, t := range tracks {
			allTracks = append(allTracks, t.Path)
		}
	}
	return allTracks, nil
}

func (m *Manager) AddSource(stationName, source string) error {
	st := m.find(stationName)
	if st == nil {
		return fmt.Errorf("station %q not found", stationName)
	}

	fullPath := source
	if !filepath.IsAbs(source) {
		fullPath = filepath.Join(m.cfg.Server.MusicDir, source)
	}
	newTracks, err := scanner.Scan(fullPath)
	if err != nil {
		return fmt.Errorf("scan source: %w", err)
	}

	st.Config.Sources = append(st.Config.Sources, source)
	st.TrackCount += len(newTracks)

	allTracks, _ := m.resolveSources(st.Config.Sources)
	st.playlist = playlist.New(allTracks)
	st.streamer = streamer.New(st.playlist, st.Name)

	return m.saveConfig()
}

func (m *Manager) RemoveSource(stationName string, index int) error {
	st := m.find(stationName)
	if st == nil {
		return fmt.Errorf("station %q not found", stationName)
	}
	if index < 0 || index >= len(st.Config.Sources) {
		return fmt.Errorf("source index %d out of range", index)
	}

	st.Config.Sources = append(st.Config.Sources[:index], st.Config.Sources[index+1:]...)

	allTracks, _ := m.resolveSources(st.Config.Sources)
	st.TrackCount = len(allTracks)
	st.playlist = playlist.New(allTracks)
	st.streamer = streamer.New(st.playlist, st.Name)

	return m.saveConfig()
}

func (m *Manager) saveConfig() error {
	for _, st := range m.stations {
		for i, sc := range m.cfg.Stations {
			if sc.Name == st.Name {
				m.cfg.Stations[i].Sources = st.Config.Sources
				break
			}
		}
	}
	return config.Save(m.configPath, m.cfg)
}
```

This is getting quite involved with the type reconciliation. Let me instead just write the files correctly and let the plan be the plan. The plan should describe what to do. Let me simplify Task 8 to just a few clean steps that write the final versions.

- [ ] **Step 3: Write final main.go**

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"radio/internal/station"
	"radio/internal/web"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	mgr, err := station.NewManager(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	handler := web.NewHandler(mgr, mgr.Port(), mgr.MusicDir())

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	// Register stream routes
	for _, st := range mgr.AllStations() {
		streamer := mgr.StreamerFor(st.Name)
		if streamer != nil {
			mux.Handle("/stream"+st.Mount, streamer)
		}
	}

	addr := fmt.Sprintf(":%d", mgr.Port())
	log.Printf("Radio server starting on http://localhost%s", addr)
	for _, st := range mgr.AllStations() {
		log.Printf("  %s: http://localhost%s/stream%s", st.Name, addr, st.Mount)
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

This requires:
- `station.Manager` has `Port()`, `MusicDir()`, `AllStations()`, `StreamerFor()`, `StationList()`, `Find()`, `AddSource()`, `RemoveSource()`
- `web.NewHandler` accepts what `station.Manager` provides
- `StationProvider` interface with `Name()`, `Mount()`, `TrackCount()`, `Sources()`

OK, this is getting complex to describe step-by-step. Let me just write the complete, correct plan file now. The key insight is: the plan should describe WHAT to write, with code blocks that work. Let me simplify by having web import station and use the concrete types.

Let me rewrite the full plan file now with correct, internally consistent Go code.<｜end▁of▁thinking｜>I realize the station manager interface reconciliation is getting tangled in the plan. Let me rewrite Task 8 cleanly to make everything type-consistent.

<｜｜DSML｜｜tool_calls>
<｜｜DSML｜｜invoke name="edit">
<｜｜DSML｜｜parameter name="path" string="true">docs/plans/2025-07-14-radio-streaming-server.md