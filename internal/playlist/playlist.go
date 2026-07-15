package playlist

import (
	"math/rand"
	"time"
)

// DefaultTrackDuration is the assumed duration per track for virtual clock.
const DefaultTrackDuration = 3 * time.Minute

type Playlist struct {
	tracks    []string
	current   int
	startTime time.Time
	trackDur  time.Duration
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
		current:   -1,
		startTime: start,
		trackDur:  DefaultTrackDuration,
	}
}

func (p *Playlist) StartTime() time.Time { return p.startTime }

// SyncToVirtual jumps the playlist to the track that should be playing
// based on elapsed time since the station started.
// Returns the track path and the fraction (0.0–1.0) into the track.
func (p *Playlist) SyncToVirtual() (string, float64) {
	if len(p.tracks) == 0 {
		return "", 0
	}
	elapsed := time.Since(p.startTime)
	trackIdx := int(elapsed/p.trackDur) % len(p.tracks)
	// Fraction within this track's time slot
	slotElapsed := elapsed % p.trackDur
	frac := float64(slotElapsed) / float64(p.trackDur)

	// Set current just before so Next() returns the virtual track
	p.current = (trackIdx - 1 + len(p.tracks)) % len(p.tracks)
	return p.tracks[trackIdx], frac
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
