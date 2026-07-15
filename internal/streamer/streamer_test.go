package streamer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func createSilentMP3(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	frame := []byte{
		0xFF, 0xFB, 0x90, 0x00,
	}
	frame = append(frame, make([]byte, 413)...)
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

func (f *fakePlaylist) SyncToVirtual() (string, float64) {
	if len(f.tracks) == 0 {
		return "", 0
	}
	return f.tracks[0], 0
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

	pr, pw := io.Pipe()
	req := httptest.NewRequest("GET", "/stream/test", nil)
	shim := &responseWriterShim{w: pw, header: make(http.Header)}

	go func() {
		srv.ServeHTTP(shim, req)
	}()

	// Read a byte to trigger header write
	buf := make([]byte, 1)
	pr.Read(buf)
	pr.Close()

	contentType := shim.Header().Get("Content-Type")
	if contentType != "audio/mpeg" {
		t.Errorf("Content-Type = %s, want audio/mpeg", contentType)
	}
	icyName := shim.Header().Get("icy-name")
	if icyName != "Test Station" {
		t.Errorf("icy-name = %s, want Test Station", icyName)
	}
	if shim.code != http.StatusOK {
		t.Errorf("status = %d, want 200", shim.code)
	}
}

func TestStreamerSendsAudioData(t *testing.T) {
	dir := t.TempDir()
	path := createSilentMP3(t, dir, "song.mp3")

	fp := &fakePlaylist{tracks: []string{path}}
	srv := New(fp, "Radio")

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

type responseWriterShim struct {
	w      io.WriteCloser
	header http.Header
	code   int
}

func (s *responseWriterShim) Header() http.Header        { return s.header }
func (s *responseWriterShim) Write(b []byte) (int, error) { return s.w.Write(b) }
func (s *responseWriterShim) WriteHeader(code int)        { s.code = code }
