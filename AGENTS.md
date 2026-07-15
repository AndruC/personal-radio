# AGENTS.md

## Project Overview

Radio is a Go server that scans local music folders and streams them as SHOUTcast/ICY-compatible internet radio stations. Each station is a continuous, clock-driven broadcast — tuning in with VLC drops you mid-song like real radio. A minimal htmx web UI manages station sources.

## Build & Test

```bash
go build -o radio.exe .
go test ./internal/... -v
```

**Requires Go 1.22+** (tested on 1.26).

## Run

```bash
go run . -config config.yaml
```

Default config path is `config.yaml`. See `config.yaml` for the format.

## Architecture

```
radio/
├── main.go                         # Entry point, flag parsing, HTTP mux
├── config.yaml                     # Default station config
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML config struct, Load/Save
│   │   └── config_test.go
│   ├── scanner/
│   │   ├── scanner.go              # Walk dirs, index MP3/OGG files
│   │   └── scanner_test.go
│   ├── playlist/
│   │   ├── playlist.go             # Shuffled track queue with virtual clock
│   │   └── playlist_test.go
│   ├── streamer/
│   │   ├── streamer.go             # SHOUTcast ICY HTTP handler
│   │   └── streamer_test.go
│   ├── station/
│   │   ├── manager.go              # Station lifecycle, source CRUD, config persistence
│   │   └── manager_test.go
│   └── web/
│       ├── handlers.go             # htmx route handlers
│       ├── handlers_test.go
│       └── templates/              # Go html/template files
│           ├── base.html           # Layout, CSS, nav
│           ├── dashboard.html      # Station list + create form
│           ├── station.html        # Station detail, source management, partials
│           └── library.html        # Music library browser + search
```

## Key Design Decisions

### Clock-driven broadcast (no idle cost)
- Each station's playlist has a `startTime` and assumes 3-minute tracks (`DefaultTrackDuration`).
- When a listener connects, `SyncToVirtual()` calculates which track should be playing based on elapsed wall-clock time, and how far into it we should be (fraction).
- The streamer seeks into the file proportionally and starts streaming.
- No goroutines, timers, or file I/O when nobody is listening. Zero idle cost.

### Streaming: firehose to VLC buffer
- Audio data is sent at full localhost speed, not throttled to playback rate.
- VLC buffers the entire playlist within seconds, then plays back at real time.
- The server goes idle immediately after sending — zero resource usage during playback.
- Logs reflect connections, not individual tracks (the server doesn't know what VLC is playing).

### No transcoding
- MP3 and OGG pass through as-is. Other formats are skipped.
- Content-Type is always `audio/mpeg` (SHOUTcast convention).
- Seeking mid-file drops you on an arbitrary byte; VLC's decoder finds the next valid frame header.

### Config persistence
- All mutations (add/remove source, create station) write through to `config.yaml` immediately.
- Source paths are normalized: directory sources get a trailing separator, pasted paths get quotes/whitespace stripped, path existence is validated before accept.

### htmx partials
- Source add/remove returns only the `#sources-list` partial, not the full page.
- Errors render inline as a red bar within the partial so htmx swaps cleanly.
- The "Add Source" input clears only on success (`hx-on::after-request`).

### Go patterns
- No global state. The `station.Manager` owns all state and is passed to handlers.
- `StationProvider` interface exposes read-only station views to templates.
- `PlaylistProvider` interface lets streamer test with fakes.
- Tests use `t.TempDir()` for all file operations — no test pollution.

## Dependencies

| Package | Purpose |
|---|---|
| `gopkg.in/yaml.v3` | Config parsing |
| `github.com/dhowden/tag` | (planned) ID3 tag reading |
| Standard library | Everything else |

## Conventions

- All paths use `filepath` for Windows compatibility.
- Error handling: skip unreadable files, log warnings, never crash the stream.
- Commit messages: conventional-ish — `feat:`, `fix:`, `style:`, `chore:`, `docs:`.
- TDD: write failing test → implement → verify pass → commit.
