-- name: GetRenditionById :one
SELECT * FROM renditions WHERE id = $1 LIMIT 1;

-- name: GetRenditionByName :one
SELECT * FROM renditions WHERE name = $1 LIMIT 1;

-- name: GetAllRenditions :many
SELECT * FROM renditions;

-- name: DeleteRenditionById :exec
DELETE FROM renditions WHERE id = $1;

-- name: CreateRendition :one
INSERT INTO renditions (
  name,
  width,
  height,
  video_bitrate,
  audio_bitrate,
  video_codec,
  audio_codec,
  fps
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: DeleteRendition :exec
DELETE FROM renditions WHERE id = $1;
