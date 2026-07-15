package playlist

import (
	"math/rand"
	"time"
)

// DefaultTrackDuration is the assumed duration per track for virtual clock.
const DefaultTrackDuration = 3 * time.Minute

type Playlist struct {
	tracks    []string
	startTime time.Time
}

func New(tracks []string) *Playlist {
	return NewWithStartTime(tracks, time.Now())
}

func NewWithStartTime(tracks []string, start time.Time) *Playlist {
	shuffled := make([]string, len(tracks))
	copy(shuffled, tracks)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return &Playlist{
		tracks:    shuffled,
		startTime: start,
	}
}

func (p *Playlist) Tracks() []string      { return p.tracks }
func (p *Playlist) StartTime() time.Time  { return p.startTime }
func (p *Playlist) Len() int              { return len(p.tracks) }
