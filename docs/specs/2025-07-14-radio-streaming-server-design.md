# Radio Streaming Server — Design Spec

**Date:** 2025-07-14
**Status:** Draft

## Overview

A Go server that scans a local music folder and streams MP3/OGG files as continuous "radio stations" via SHOUTcast/Icecast-compatible HTTP streams. VLC Media Player connects as the client. Multiple stations can be configured, each drawing from different folders. A minimal htmx-based web UI manages station sources.

## Goals

- Stream continuous, randomized music from a local collection to VLC over HTTP
- Support multiple simultaneous stations, each with its own set of source folders/files
- No transcoding — MP3 and OGG pass through as-is; other formats are skipped
- Minimal web UI (htmx) to add/remove sources from stations
- Single binary, no external streaming server dependency
- Single listener assumed (embedded SHOUTcast server, not external Icecast)

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌────────────────┐
│  Web UI     │────▶│  Station     │────▶│  Playlist      │
│  (htmx)     │     │  Manager     │     │  Engines (N)   │
└─────────────┘     └──────────────┘     └───────┬────────┘
                                                 │
┌─────────────┐     ┌──────────────┐     ┌───────▼────────┐
│  Config     │────▶│  Music       │     │  SHOUTcast     │
│  File (yaml)│     │  Scanner     │     │  Streamer (N)  │
└─────────────┘     └──────────────┘     └────────────────┘
```

### Components

**Config File** (`config.yaml`)
- Defines server port, music directory, and stations
- Each station has a name, mount point, and a list of source paths (folders or individual files)
- Edits from the web UI write back to this file

**Music Scanner**
- Walks the configured music directory and indexes all MP3 and OGG files
- Extracts filename and, where available, ID3/Vorbis tags for display name
- Resolves source paths: directories are recursively scanned for eligible files; individual files are added directly

**Station Manager**
- Owns the config in memory
- Starts/stops playlist engines and streamers per station
- Persists config changes from the web UI back to disk

**Playlist Engine** (one per station)
- Holds the resolved track list from all sources
- Shuffles tracks on start, advances to next when current ends, loops
- Signals the streamer when a new track begins

**SHOUTcast Streamer** (one per station)
- Serves an ICY-compatible HTTP stream at `/stream/:mount`
- Reads MP3/OGG frames from the current track and writes to connected clients
- Handles track transitions (brief silence gap between tracks)
- Sends ICY metadata (now-playing title) to compatible clients

**Web UI** (htmx, minimal)
- Dashboard (`GET /`) — station list with now-playing status
- Station detail (`GET /stations/:name`) — sources, tracks, remove sources
- Library browser (`GET /library`) — search/filter scanned files, add to station
- Add source form (`POST /stations/:name/sources`)
- Remove source (`DELETE /stations/:name/sources/:id`)
- Status poll (`GET /stations/:name/status`) — htmx auto-refresh target

## Data Model

```yaml
# config.yaml
server:
  port: 8080
  music_dir: "P:\\My Media\\My Music"

stations:
  - name: "Rock"
    mount: "/rock"
    sources:
      - "Rock"
      - "Classic Rock"
      - "favorites/song.mp3"
  - name: "Jazz"
    mount: "/jazz"
    sources:
      - "Jazz"
```

- Source paths are relative to `music_dir` unless absolute
- Sources are resolved at startup and when config changes
- Each source is checked: if directory → recurse for MP3/OGG; if file (MP3/OGG) → add directly; otherwise skip

## Streaming Flow

```
Playlist Engine picks next track
  → opens file
  → reads frames
  → writes to ring buffer
  → SHOUTcast handler reads from buffer
  → client (VLC) consumes ICY HTTP stream
```

Track transitions: current track plays to end, brief silence gap (no crossfade in v1), next track begins. Client stays connected.

## Error Handling

| Scenario | Behavior |
|---|---|
| Missing/moved file | Skip in playlist, log warning, flag in UI |
| Corrupt or unsupported file | Skip, log, don't crash stream |
| Empty station | Stream silence |
| Corrupt config on startup | Refuse to start, log parse error |
| Config save failure | Return error in UI, keep in-memory state intact |

## Non-Goals (v1)

- Transcoding (FLAC, AAC, WMA, etc.)
- Authentication or user management
- Multiple simultaneous listeners (embedded SHOUTcast, not external Icecast)
- Crossfade or gapless playback
- Smart shuffling (weighted, anti-repeat logic)
- Cover art or rich media metadata
- Hot config reload (restart-required for manual config edits)

## Routes

| Method | Route | Purpose |
|---|---|---|
| GET | `/` | Dashboard — station list, now-playing |
| GET | `/stations/:name` | Station detail — sources, tracks |
| GET | `/library` | Browse/search music library |
| POST | `/stations/:name/sources` | Add source to station |
| DELETE | `/stations/:name/sources/:id` | Remove source from station |
| GET | `/stations/:name/status` | htmx poll: now-playing info |
| GET | `/stream/:mount` | SHOUTcast/ICY audio stream |

## Testing Strategy

- **Unit tests:** config parsing, playlist shuffle/loop, source resolution (file vs folder), ICY header generation, scanner
- **Integration test:** start server with test MP3/OGG files, verify stream is playable by an HTTP client that speaks ICY
- **UI:** manual testing acceptable for v1

## Dependencies (Go)

- `gopkg.in/yaml.v3` — config parsing
- `net/http` — HTTP server and SHOUTcast streaming
- `github.com/dhowden/tag` — ID3/Vorbis tag reading (for display names)
- Standard library for file walking, ring buffer, etc.

## Future Considerations

- Smart shuffle (anti-repeat, weighted by recent plays)
- Crossfade between tracks
- External Icecast server support for multiple listeners
- Richer web UI: drag-and-drop, cover art, playback history
- Transcoding support for additional formats
- Hot config reload
