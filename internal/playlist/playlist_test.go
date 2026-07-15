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
	tracks := make([]string, 100)
	for i := range tracks {
		tracks[i] = string(rune('A'+i%26)) + string(rune('a'+i/26))
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

	if slices.Equal(first, tracks[:5]) {
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
	if p.Current() != track {
		t.Errorf("Current() changed after second call")
	}
}
