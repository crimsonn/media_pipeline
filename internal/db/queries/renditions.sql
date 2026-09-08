-- name: GetRenditionById :one
SELECT * FROM renditions WHERE id = ? LIMIT 1;

-- name: GetRenditionByName :one
SELECT * FROM renditions WHERE name = ? LIMIT 1;

-- name: CreateRendition :one
INSERT INTO renditions(
  name,
  width,
  height,
  video_bitrate,
  audio_bitrate,
  video_codec,
  audio_codec,
  fps,
  created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteRendition :exec
DELETE FROM renditions WHERE id = ?;
