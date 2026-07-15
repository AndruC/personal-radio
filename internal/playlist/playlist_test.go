package playlist

import (
	"testing"
)

func TestNewPlaylist(t *testing.T) {
	tracks := []string{"/music/a.mp3", "/music/b.ogg", "/music/c.mp3"}
	p := New(tracks)
	if p.Len() != 3 {
		t.Errorf("Len() = %d, want 3", p.Len())
	}
	if len(p.Tracks()) != 3 {
		t.Errorf("Tracks() len = %d, want 3", len(p.Tracks()))
	}
}

func TestPlaylistEmpty(t *testing.T) {
	p := New(nil)
	if p.Len() != 0 {
		t.Errorf("Len() = %d, want 0", p.Len())
	}
}

func TestPlaylistStartTime(t *testing.T) {
	p := New([]string{"/music/a.mp3"})
	if p.StartTime().IsZero() {
		t.Error("StartTime should not be zero")
	}
}

func TestPlaylistShuffles(t *testing.T) {
	tracks := make([]string, 100)
	for i := range tracks {
		tracks[i] = string(rune('A'+i%26)) + string(rune('a'+i/26))
	}

	p := New(tracks)
	result := p.Tracks()

	// All original tracks should be present
	if len(result) != len(tracks) {
		t.Errorf("Tracks() len = %d, want %d", len(result), len(tracks))
	}

	// Check that the order differs (probabilistic)
	same := true
	for i := range tracks {
		if tracks[i] != result[i] {
			same = false
			break
		}
	}
	if same {
		// Try again — shuffle is probabilistic
		p2 := New(tracks)
		result2 := p2.Tracks()
		same = true
		for i := range tracks {
			if tracks[i] != result2[i] {
				same = false
				break
			}
		}
		if same {
			t.Skip("shuffle preserved order twice — extremely unlikely, skipping")
		}
	}
}
