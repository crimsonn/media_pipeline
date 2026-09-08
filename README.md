# media pipeline

A small local lab for learning **ffmpeg** and a **watch-folder → transcode → HLS** pipeline.

Drop a video into `watch/`. A watchdog notices it, waits until the file stops growing, then asks a gRPC transcoder to encode multiple renditions with ffmpeg. The result is an HLS VOD package under `done/`.

This is not a production encoder. It is a playground for how adaptive bitrate streaming is assembled.

## What you get

```
watch/clip.mp4
        │
        ▼
   watchdog (polls folder)
        │  gRPC TranscodeVideo
        ▼
   transcoder (worker pool)
        │  one ffmpeg process per rendition
        ▼
done/clip/
  master.m3u8          ← player entry point (picks 1080p or 720p)
  1080p/manifest.m3u8  ← 6s MPEG-TS segments
  720p/manifest.m3u8
```

Default renditions (hardcoded in the watchdog):

| name  | size      | video bitrate | audio bitrate |
|-------|-----------|---------------|---------------|
| 1080p | 1920×1080 | 5 Mbps        | 128 kbps AAC  |
| 720p  | 1280×720  | 2 Mbps        | 128 kbps AAC  |

After a successful job the source file is **deleted** from `watch/`.

## How the pipeline works

Two processes, one gRPC contract (`TranscoderService.TranscodeVideo` in `api/proto/transcoder/v1/transcoder.proto`).

### 1. Watchdog — ingest

`cmd/watchdog` polls `WATCHDOG_FOLDER` (default `./watch`) every few seconds.

- New filenames are queued (capacity `QUEUE_CAPACITY`).
- A worker watches `mtime`. If the file has not changed for two consecutive checks, it is treated as fully written (so a copy-in-progress is not transcoded).
- It then opens a streaming RPC to the transcoder with the source path, output dir (`WATCHDOG_OUTPUT`, default `./done`), and the rendition list.

### 2. Transcoder — encode

`cmd/transcoder` listens on `TRANSCODER_ADDR` (default `:50051`).

- Each requested resolution becomes a `Task` on a worker pool.
- A worker creates `done/<name>/<rendition>/` and runs ffmpeg.
- When every rendition finishes, it writes `master.m3u8` and removes the source file.
- Progress/result is sent back on the gRPC stream (`IN_PROGRESS` / `COMPLETED` / `FAILED`).

Workers are independent, so 1080p and 720p encode in parallel (subject to CPU).

## What ffmpeg is doing

Each worker runs roughly this (see `renditionArgs` in `services/transcoder/workers.go`):

```text
ffmpeg -i source.mp4
  -map 0:v:0
  -vf scale=w=1920:h=1080
  -c:v libx264 -profile:v high -level 4.0 -preset veryfast
  -b:v 5000000 -maxrate 5500000 -bufsize 10000000
  -force_key_frames expr:gte(t,n_forced*6) -sc_threshold 0
  -map 0:a:0? -c:a aac -b:a 128000 -ac 2
  -f hls -hls_time 6 -hls_playlist_type vod -hls_flags independent_segments
  -hls_segment_filename .../seg_%05d.ts
  .../manifest.m3u8
```

Useful pieces:

| flag | why it is here |
|------|----------------|
| `-vf scale=` | resize this rendition |
| `libx264` + `veryfast` | H.264 encode, faster preset for a local test |
| `-b:v` / `-maxrate` / `-bufsize` | CBR-ish ABR ladder (maxrate ~110% of target, 2s VBV buffer) |
| `-force_key_frames` every 6s | GOP aligned to segment length so a player can switch renditions at segment boundaries |
| `-sc_threshold 0` | no extra scene-cut keyframes (keeps GOPs regular) |
| `-map 0:a:0?` | take first audio if it exists (`?` = optional) |
| `-f hls` + `-hls_time 6` | MPEG-TS segments of ~6 seconds |
| `-hls_playlist_type vod` | finite playlist with `#EXT-X-ENDLIST` |
| `independent_segments` | each `.ts` starts on a keyframe |

The **master playlist** is written in Go, not by ffmpeg. It lists each variant with bandwidth, resolution, and codecs so a player can pick a ladder step:

```text
#EXTM3U
#EXT-X-VERSION:6
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-STREAM-INF:BANDWIDTH=5128000,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"
1080p/manifest.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2128000,RESOLUTION=1280x720,CODECS="avc1.640028,mp4a.40.2"
720p/manifest.m3u8
```

Play `done/<name>/master.m3u8` in anything that speaks HLS (VLC, ffplay, a browser with hls.js).

## Layout

```text
cmd/transcoder/          gRPC server entrypoint
cmd/watchdog/            folder watcher + gRPC client
services/transcoder/     ffmpeg workers + TranscodeVideo handler
services/watchdog/       poll loop, stability checks, RPC
api/proto/transcoder/v1/ TranscoderService protobuf
pkg/pb/                  generated Go stubs
internal/config/         env helpers (defaults if unset)
watch/                   drop source files here (gitignored)
done/                    HLS output (gitignored)
Dockerfile               multi-stage build (watchdog + transcoder)
docker-compose.yml       shared watch/done volumes
```

## Prerequisites

- Go (see `go.mod`)
- [ffmpeg](https://ffmpeg.org/) on `PATH` (with `libx264`)
- `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` only if you change the `.proto`

## Run

```bash
mkdir -p watch done

# terminal 1
go run ./cmd/transcoder

# terminal 2
go run ./cmd/watchdog

# then copy a file in
cp /path/to/clip.mp4 watch/
```

Wait until the watchdog decides the file is stable, then check `done/<basename>/master.m3u8`.

### Docker

You do **not** copy files into the container. `./watch` on your machine is bind-mounted into both services at `/data/watch`, and `./done` at `/data/done`. The watchdog only sees whatever you drop on the host.

```bash
mkdir -p watch done
docker compose up --build
cp /path/to/clip.mp4 watch/
```

HLS shows up in `./done/<basename>/` on the host (play `master.m3u8` with VLC). After a successful job the file is removed from `watch/`.

Both containers share those folders because the transcoder runs ffmpeg on the **path the watchdog sends** (`/data/watch/clip.mp4`). If the mounts did not match, ffmpeg would look for a file that only exists in the other container.

Config is read from the process environment (`internal/config`). Defaults match a local run; copy `.env.example` if you want a reminder of the knobs. Docker Compose sets its own env in `docker-compose.yml`.

| variable | default | used by |
|----------|---------|---------|
| `TRANSCODER_ADDR` | `:50051` | both (watchdog dials `transcoder:50051` in Compose) |
| `WATCHDOG_FOLDER` | `./watch` | watchdog (`/data/watch` in Compose) |
| `WATCHDOG_OUTPUT` | `./done` | watchdog (`/data/done` in Compose) |
| `WATCHDOG_DELAY` | `5s` | parsed at startup |
| `NUM_WORKERS` | `10` | watchdog queue workers |
| `QUEUE_CAPACITY` | `100` | watchdog task channel |
| `TRANSCODER_WORKERS` | (example only) | not wired yet; transcoder currently uses 10 workers in code |

Regenerate protobufs after editing the API:

```bash
make proto
```

## Notes while learning

- One **input**, N **ffmpeg processes**, one **master playlist** is the usual ABR pattern.
- Segment duration and keyframe interval should match; otherwise HLS switches can glitch.
- The watchdog’s “file stopped changing” check is a simple substitute for a real upload-complete signal.
- Source deletion on success is convenient for a lab and dangerous if you still need the original.

## Next steps

This file is also a progress log. The pipeline already accepts a list of renditions on `TranscodeVideoRequest`; the watchdog just hardcodes 1080p + 720p in `services/watchdog/orchestrator.go`. Next is to treat that list as a **named profile** you edit in a UI and persist, instead of a literal in code.

### Encoding profiles

A profile is a reusable ABR ladder: name + ordered renditions (`name`, `width`, `height`, `video_bps`, `audio_bps` — same fields as `TranscodeResolution`). Examples: `web-default`, `mobile`, `archive-1080`.

- Store profiles in a database (SQLite is enough for a local lab; Postgres if you want it in Compose).
- Watchdog (or a small API in front of it) loads a profile by id and passes `resolutions` through gRPC. ffmpeg does not change; only where the ladder comes from does.
- Keep one “default” profile so drop-in-`watch/` still works without clicking through the UI.

### UI

A simple app to:

- CRUD profiles and their renditions (add 480p, drop 1080p, tweak bitrates).
- See jobs: file in, profile used, state, link to `done/<name>/master.m3u8`.
- Optionally pick a profile per job, or set which profile the watchdog applies to new files.

The UI talks to an HTTP API; the transcoder stays gRPC + ffmpeg.

### Later (when the above works)

- Wire `TRANSCODER_WORKERS` and real ffmpeg `%` progress on the stream.
- Play HLS in the browser (hls.js) instead of only VLC.
- Stop deleting the source until you choose to; keep originals under `watch/` or an archive folder.
