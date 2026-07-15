package streamer

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type PlaylistProvider interface {
	Next() (string, bool)
	Current() string
	SyncToVirtual() (string, float64)
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

	log.Printf("[%s] listener connected", s.name)

	// Sync to virtual broadcast position, seek into the track
	trackPath, frac := s.playlist.SyncToVirtual()
	if trackPath != "" {
		log.Printf("[%s] virtual track: %s (%.0f%%) ", s.name, filepath.Base(trackPath), frac*100)
		if err := s.streamFileSeek(w, trackPath, frac, canFlush, flusher); err != nil {
			if !isClientDisconnect(err) {
				log.Printf("[%s] stream error: %v", s.name, err)
			}
			return
		}
		// Advance past the virtual track so the loop picks up the next one
		s.playlist.Next()
	} else {
		log.Printf("[%s] no tracks in playlist", s.name)
	}

	for {
		trackPath, ok := s.playlist.Next()
		if !ok {
			log.Printf("[%s] playlist ended", s.name)
			return
		}

		log.Printf("[%s] now playing: %s", s.name, filepath.Base(trackPath))
		err := s.streamFile(w, trackPath, canFlush, flusher)
		if err != nil {
			if !isClientDisconnect(err) {
				log.Printf("[%s] stream error: %v", s.name, err)
			} else {
				log.Printf("[%s] client disconnected", s.name)
			}
			return
		}
		log.Printf("[%s] track finished, advancing", s.name)
	}
}

func (s *Streamer) streamFileSeek(w io.Writer, path string, frac float64, canFlush bool, flusher http.Flusher) error {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("cannot open %s: %v", path, err)
		return nil
	}
	defer f.Close()

	// Seek into the file based on virtual time fraction
	if frac > 0 {
		info, err := f.Stat()
		if err == nil && info.Size() > 0 {
			offset := int64(float64(info.Size()) * frac)
			// Align to a sensible boundary
			offset = offset - (offset % 4096)
			f.Seek(offset, io.SeekStart)
		}
	}

	return s.readFile(w, f, canFlush, flusher)
}

func (s *Streamer) streamFile(w io.Writer, path string, canFlush bool, flusher http.Flusher) error {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("cannot open %s: %v", path, err)
		return nil
	}
	defer f.Close()

	return s.readFile(w, f, canFlush, flusher)
}

func (s *Streamer) readFile(w io.Writer, f *os.File, canFlush bool, flusher http.Flusher) error {
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
