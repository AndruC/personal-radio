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
