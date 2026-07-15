# Radio

A Go server that turns your local music collection into internet radio stations. Stream MP3 and OGG files to VLC as continuous broadcasts — tune in mid-song like real radio.

## How it works

- Scans folders for MP3/OGG files
- Each station runs on a virtual clock — songs "play" in the background even when nobody's listening
- When you tune in with VLC, you drop into the middle of whatever's "on air"
- A web dashboard lets you manage stations and add/remove music sources
- Zero CPU or disk usage when nobody's listening

### Streaming behavior

The server sends audio at full localhost speed — VLC buffers the entire playlist within seconds, then plays it back at real time. This means:

- The server's work is done almost instantly after you connect
- Console logs show one connection per tune-in, not per-track
- VLC handles gapless sequential playback from its buffer
- The server stays idle while you listen, consuming no resources

## Quick start

1. Edit `config.yaml` to point at your music folders
2. `go run .`
3. Open `http://localhost:8080` in your browser
4. In VLC: Media → Open Network Stream → `http://localhost:8080/stream/your-mount`

## Requirements

- Go 1.22+
- Music in MP3 or OGG format
- VLC (or any SHOUTcast/ICY-compatible player)

## Config

Edit `config.yaml` to point at your music. Sources can be folder paths (recursively scanned for MP3/OGG) or individual files. Paths are relative to `music_dir` unless they start with a drive letter.

```yaml
server:
  port: 8080
  music_dir: "C:\\Users\\You\\Music"

stations:
  - name: "Rock"
    mount: "/rock"
    sources:
      - "Rock"                         # C:\Users\You\Music\Rock\
      - "Classic Rock"                 # C:\Users\You\Music\Classic Rock\
      - "D:\\Other Stuff\\favorites"   # Absolute path on another drive
      - "misc\\single.mp3"             # Individual file
  - name: "Electronic"
    mount: "/electronic"
    sources:
      - "Electronic"
```

Stations and sources can also be managed from the web UI — no config editing needed after initial setup.
