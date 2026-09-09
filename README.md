# media pipeline

A local lab for learning **ffmpeg**, **HLS**, and a **watch-folder → database queue → worker** transcode pipeline.

Drop a video into `watch/`. A watcher waits until the file stops growing, then records it as pending. You pick a **transcode profile** in the UI. Postgres holds the job (and one task per rendition). Transcoder workers claim those tasks and run ffmpeg. HLS lands under `done/`.

This is not a production encoder. It is a playground for how adaptive bitrate streaming is assembled.

gRPC is gone. Services do not call each other. They share **PostgreSQL**.

## What you get

```
watch/clip.mp4
        │
        ▼
   watcher (polls folder, waits until mtime is stable)
        │  INSERT pending_files
        ▼
   UI / API  — assign a transcode profile
        │  INSERT jobs + job_tasks (one row per rendition)
        ▼
   Postgres  — the queue
        │  ClaimJobTask (FOR UPDATE SKIP LOCKED)
        ▼
   transcoder workers  — one ffmpeg process per claimed task
        ▼
done/clip/
  master.m3u8          ← player entry point
  1080p/manifest.m3u8  ← ~6s MPEG-TS segments (duration comes from the profile)
  720p/manifest.m3u8
```

Renditions and ladders are not hardcoded. You create **renditions** (size, bitrates, codecs) and group them into **profiles** (named ladder + HLS segment time). The watcher only discovers files; encoding starts when a profile is assigned.

After a successful job the source file is **deleted** from `watch/`.

## How the pipeline works

Four processes plus Postgres.

### 1. Watcher — ingest

`cmd/watcher` polls `WATCH_DIRECTORY` (default `./watch`).

- New filenames go on an in-process queue (`NUM_WORKERS`, `QUEUE_CAPACITY`).
- A worker watches `mtime`. After two consecutive checks with no change, the file is treated as fully written.
- It inserts a `pending_files` row (`status = waiting`). It does **not** start ffmpeg.

### 2. API + UI — profiles and enqueue

`cmd/api` is a Gin server (`API_ADDR`:`API_PORT`, default `localhost:8080`).

The TanStack UI in `web/` (dev server on port 3000) talks to `/api/v1`:

| | |
|---|---|
| Renditions | `GET/POST /transcoder/renditions`, `DELETE /transcoder/renditions/:id` |
| Profiles | `GET/POST /transcoder/profiles`, `GET /transcoder/profiles/:id` |
| Pending | `GET /pending/files`, `POST /pending/files` (assign profile, enqueue) |
| Jobs | `GET /jobs/latest` |
| Playback | `GET /playback/outputs`, `GET /playback/hls/*` |

`POST /pending/files` calls `EnqueueTranscode`: one `jobs` row plus one `job_tasks` row per rendition on that profile.

### 3. Postgres — queue and source of truth

Schema lives in `internal/db/schema.sql` (applied on process startup and by Compose init). sqlc generates Go from `internal/db/queries/*.sql`.

| table | role |
|-------|------|
| `renditions` | reusable encode settings (name, WxH, bitrates, codecs, fps) |
| `transcode_profiles` | named ladder + `hls_segment_time` |
| `profile_renditions` | which renditions belong to a profile, and order |
| `pending_files` | files the watcher saw, waiting for a profile |
| `jobs` | one transcode of one file with a profile (`pending` / `running` / `completed` / `failed`) |
| `job_tasks` | one ffmpeg run per rendition; this is the work queue |

Workers claim work with `FOR UPDATE SKIP LOCKED` so two processes cannot take the same task.

### 4. Transcoder — encode

`cmd/transcoder` is a worker pool (`TRANSCODER_WORKERS`, default 10). Every ~2s each worker tries `ClaimJobTask`. On a hit it:

- marks the parent job `running`
- creates `done/<stem>/<rendition>/`
- runs ffmpeg
- marks the task completed or failed
- when every task on the job is done: writes `master.m3u8`, marks the job completed, deletes the source

Workers are independent, so renditions on the same job encode in parallel (subject to CPU).

## What ffmpeg is doing

Each claimed task runs roughly this (see `renditionArgs` in `internal/transcoder/worker.go`). Bitrate values on the rendition are passed as kilobits (`5000` → `-b:v 5000k`). Segment length comes from the profile.

```text
ffmpeg -threads 2 -i source.mp4
  -map 0:v:0
  -vf scale=w=1920:h=1080
  -c:v libx264 -profile:v high -level 4.0 -preset veryfast
  -b:v 5000k -maxrate 5500k -bufsize 10000k
  -force_key_frames expr:gte(t,n_forced*6) -sc_threshold 0
  -map 0:a:0? -c:a aac -b:a 128k -ac 2
  -f hls -hls_time 6 -hls_playlist_type vod -hls_flags independent_segments
  -hls_segment_filename .../seg_%05d.ts
  .../manifest.m3u8
```

| flag | why it is here |
|------|----------------|
| `-vf scale=` | resize this rendition |
| `libx264` + `veryfast` | H.264 encode, faster preset for a local test |
| `-b:v` / `-maxrate` / `-bufsize` | CBR-ish ABR ladder (maxrate ~110% of target, 2s VBV buffer) |
| `-force_key_frames` every N seconds | GOP aligned to segment length so a player can switch renditions at segment boundaries |
| `-sc_threshold 0` | no extra scene-cut keyframes (keeps GOPs regular) |
| `-map 0:a:0?` | take first audio if it exists (`?` = optional) |
| `-f hls` + `-hls_time` | MPEG-TS segments |
| `-hls_playlist_type vod` | finite playlist with `#EXT-X-ENDLIST` |
| `independent_segments` | each `.ts` starts on a keyframe |

The **master playlist** is written in Go (`internal/transcoder/master.go`), not by ffmpeg:

```text
#EXTM3U
#EXT-X-VERSION:6
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-STREAM-INF:BANDWIDTH=...,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"
1080p/manifest.m3u8
```

Play via the UI (hls.js) or open `done/<name>/master.m3u8` in VLC / ffplay.

## Layout

```text
cmd/watcher/             folder poller → pending_files
cmd/transcoder/          Postgres-backed ffmpeg workers
cmd/api/                 Gin HTTP API
internal/watcher/        stability checks, pending insert
internal/transcoder/     claim loop, ffmpeg, master playlist
internal/api/            handlers: transcoder, pending, jobs, playback
internal/db/             pool, schema apply, enqueue transaction
internal/db/schema.sql   tables
internal/db/queries/     sqlc SQL + generated Go
internal/server/         Gin router
internal/config/         env
web/                     TanStack UI (renditions, profiles, pending, jobs, playback)
Dockerfile               Go images: watcher, api, transcoder
web/Dockerfile           TanStack UI
docker-compose.yml       postgres, watcher, api, transcoder, web
watch/                   drop source files here (gitignored)
done/                    HLS output (gitignored)
```

## Prerequisites

- Go (see `go.mod`)
- [ffmpeg](https://ffmpeg.org/) on `PATH` (with `libx264`)
- PostgreSQL 17 (Compose is enough)
- [Bun](https://bun.sh/) (or Node) for `web/`
- `sqlc` only if you change SQL (`make tools` then `make sqlc`)

Processes read the **environment**, not `.env` automatically. Export vars or use a runner that injects them. Copy `.env.example` as a reminder.

## Run

### Docker (full stack)

```bash
mkdir -p watch done
docker compose up --build
```

Then open [http://localhost:3000](http://localhost:3000). API is on port 8080, Postgres on 5432.

Drop files on the **host** into `./watch` (bind-mounted into watcher, api, and transcoder at `/data/watch`). HLS appears in `./done`.

`VITE_API_URL` is built as `http://localhost:8080/api/v1` because the browser is the client, not the `web` container.

### Local (UI / Go on the host)

```bash
mkdir -p watch done
docker compose up -d postgres

# from repo root, with DATABASE_URL (and the rest) in the environment
make run-watcher
make run-transcoder
make run-api

cd web && bun install && bun run dev   # http://localhost:3000
```

`make run-all` builds `bin/` and starts watcher, transcoder, and api together via goreman (`Procfile`).

Then:

1. Create renditions and a profile in the UI.
2. Copy a file into `watch/`.
3. When it appears under **Pending files**, assign the profile.
4. Watch **Home** for job/task status; play the result under **Playback**.

### Config

| variable | default | used by |
|----------|---------|---------|
| `DATABASE_URL` | `postgres://media:media@localhost:5432/media_pipeline?sslmode=disable` | all Go services |
| `API_ADDR` / `API_PORT` | `localhost` / `8080` | api |
| `ENVIRONMENT` | `development` | api (Gin mode, CORS) |
| `TRANSCODER_WORKERS` | `10` | transcoder |
| `WATCH_DIRECTORY` | `./watch` | watcher, api (enqueue path) |
| `OUTPUT_DIRECTORY` | `./done` | watcher, api, playback, jobs |
| `POLLING_INTERVAL` | `5s` | parsed at startup |
| `NUM_WORKERS` / `QUEUE_CAPACITY` | `10` / `100` | watcher in-process queue |
| `VITE_API_URL` | `http://localhost:8080/api/v1` | web (`web/.env`) |

## Notes while learning

- One **input**, N **ffmpeg processes**, one **master playlist** is the usual ABR pattern.
- Segment duration and keyframe interval should match; otherwise HLS switches can glitch.
- Postgres + `SKIP LOCKED` is a small stand-in for a dedicated queue (Redis, SQS, NATS).
- The watcher’s “file stopped changing” check is a substitute for a real upload-complete signal.
- Source deletion on success is convenient for a lab and dangerous if you still need the original.

## Next steps

This file is a progress log. Profiles, the UI, Postgres-as-queue, in-browser HLS, and Compose (including `web`) are in. Still open:

- Wire `POLLING_INTERVAL` into the watcher tickers (they are still 5s in code).
- Optional default profile so a drop in `watch/` can enqueue without the UI.
- Stop deleting the source until you choose to; keep originals under `watch/` or an archive folder.

The `web/` UI (routes, API client, profiles, pending enqueue, job feed, HLS playback) was fully wired by Claude.
