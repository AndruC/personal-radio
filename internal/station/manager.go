package station

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"radio/internal/config"
	"radio/internal/playlist"
	"radio/internal/scanner"
	"radio/internal/streamer"
)

// StationProvider is the read-only view of a station exposed to the web UI.
type StationProvider interface {
	Name() string
	Mount() string
	TrackCount() int
	Sources() []string
}

type StationInfo struct {
	name       string
	mount      string
	trackCount int
	Config     config.StationConfig
	streamer   *streamer.Streamer
	playlist   *playlist.Playlist
}

var _ StationProvider = (*StationInfo)(nil)

func (s *StationInfo) Name() string      { return s.name }
func (s *StationInfo) Mount() string     { return s.mount }
func (s *StationInfo) TrackCount() int   { return s.trackCount }
func (s *StationInfo) Sources() []string { return s.Config.Sources }

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

func (m *Manager) Port() int                  { return m.cfg.Server.Port }
func (m *Manager) MusicDir() string           { return m.cfg.Server.MusicDir }
func (m *Manager) AllStations() []*StationInfo { return m.stations }

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
		if s.name == name {
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
		name:       sc.Name,
		mount:      strings.TrimPrefix(sc.Mount, "/"),
		trackCount: len(tracks),
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

	// Strip quotes and whitespace from copy-pasted paths
	source = strings.TrimSpace(source)
	source = strings.Trim(source, "\"'")

	fullPath := source
	if !filepath.IsAbs(source) {
		fullPath = filepath.Join(m.cfg.Server.MusicDir, source)
	}

	// Validate and normalize the path
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("path not found: %s", fullPath)
	}
	displaySource := source
	if info.IsDir() {
		// Normalize: ensure directory sources end with a separator
		displaySource = strings.TrimRight(source, "\\/") + string(filepath.Separator)
	}

	newTracks, err := scanner.Scan(fullPath)
	if err != nil {
		return fmt.Errorf("scan source: %w", err)
	}
	if len(newTracks) == 0 {
		return fmt.Errorf("no MP3 or OGG files found in %s", source)
	}

	st.Config.Sources = append(st.Config.Sources, displaySource)
	st.trackCount += len(newTracks)

	allTracks, _ := m.resolveSources(st.Config.Sources)
	st.playlist = playlist.NewWithStartTime(allTracks, st.playlist.StartTime())
	st.streamer = streamer.New(st.playlist, st.name)

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
	st.trackCount = len(allTracks)
	st.playlist = playlist.NewWithStartTime(allTracks, st.playlist.StartTime())
	st.streamer = streamer.New(st.playlist, st.name)

	return m.saveConfig()
}

func (m *Manager) CreateStation(name, mount string) error {
	if m.find(name) != nil {
		return fmt.Errorf("station %q already exists", name)
	}
	sc := config.StationConfig{
		Name:    name,
		Mount:   mount,
		Sources: []string{},
	}
	if _, err := m.createStation(sc); err != nil {
		return err
	}
	m.cfg.Stations = append(m.cfg.Stations, sc)
	return config.Save(m.configPath, m.cfg)
}

func (m *Manager) StreamerForMount(mount string) *streamer.Streamer {
	mount = strings.TrimPrefix(mount, "/")
	for _, s := range m.stations {
		if s.mount == mount {
			return s.streamer
		}
	}
	return nil
}

func (m *Manager) saveConfig() error {
	for _, st := range m.stations {
		for i, sc := range m.cfg.Stations {
			if sc.Name == st.name {
				m.cfg.Stations[i].Sources = st.Config.Sources
				break
			}
		}
	}
	return config.Save(m.configPath, m.cfg)
}
