package streamer

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultTrackDuration is the assumed duration per track for virtual clock.
const DefaultTrackDuration = 3 * time.Minute

type PlaylistProvider interface {
	Tracks() []string
	StartTime() time.Time
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

	tracks := s.playlist.Tracks()
	if len(tracks) == 0 {
		log.Printf("[%s] no tracks in playlist", s.name)
		return
	}

	log.Printf("[%s] listener connected (%d tracks)", s.name, len(tracks))

	// Each connection gets its own independent cursor.
	// Calculate virtual position once and iterate from there.
	startTime := s.playlist.StartTime()
	elapsed := time.Since(startTime)
	trackIdx := int(elapsed/DefaultTrackDuration) % len(tracks)
	frac := float64(elapsed%DefaultTrackDuration) / float64(DefaultTrackDuration)

	// Play the virtual track with seek
	virtualTrack := tracks[trackIdx]
	log.Printf("[%s] virtual track: %s (%.0f%%)", s.name, filepath.Base(virtualTrack), frac*100)
	if err := s.streamFileSeek(w, virtualTrack, frac, canFlush, flusher); err != nil {
		if !isClientDisconnect(err) {
			log.Printf("[%s] stream error: %v", s.name, err)
		} else {
			log.Printf("[%s] client disconnected", s.name)
		}
		return
	}

	// Loop through remaining tracks from local cursor
	pos := trackIdx
	for {
		pos = (pos + 1) % len(tracks)
		trackPath := tracks[pos]

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

	if frac > 0 {
		info, err := f.Stat()
		if err == nil && info.Size() > 0 {
			offset := int64(float64(info.Size()) * frac)
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

// Shuffle returns a shuffled copy of the given slice.
func Shuffle(tracks []string) []string {
	shuffled := make([]string, len(tracks))
	copy(shuffled, tracks)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}
