package streamer

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type PlaylistProvider interface {
	Next() (string, bool)
	Current() string
	SyncToVirtual() string
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

func (s *Streamer) Name() string {
	return s.name
}

func (s *Streamer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("icy-name", s.name)
	w.Header().Set("icy-pub", "0")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	// Sync to virtual broadcast position, then loop normally
	s.playlist.SyncToVirtual()

	for {
		trackPath, ok := s.playlist.Next()
		if !ok {
			return
		}

		err := s.streamFile(w, trackPath, canFlush, flusher)
		if err != nil {
			if !isClientDisconnect(err) {
				log.Printf("stream error on %s: %v", trackPath, err)
			}
			return
		}
	}
}

func (s *Streamer) streamFile(w io.Writer, path string, canFlush bool, flusher http.Flusher) error {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("cannot open %s: %v", path, err)
		return nil
	}
	defer f.Close()

	buf := make([]byte, 16*1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
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

// isClientDisconnect returns true if the error is from a client disconnecting,
// which is normal and should not be logged as an error.
func isClientDisconnect(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "forcibly closed") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "closed pipe")
}
