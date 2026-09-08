-- name: GetProfileWithRenditions :many
SELECT
  p.id AS profile_id,
  p.name AS profile_name,
  p.description AS profile_description,
  p.hls_segment_time,
  r.id AS rendition_id,
  r.name AS rendition_name,
  r.width,
  r.height,
  r.video_bitrate,
  r.audio_bitrate,
  r.video_codec,
  r.audio_codec,
  pr.stream_index
FROM transcode_profiles AS p
JOIN profile_renditions AS pr
  ON p.id = pr.profile_id
JOIN renditions AS r
  ON r.id = pr.rendition_id
WHERE
  p.id = ?
ORDER BY
  pr.stream_index;

-- name: GetTranscodeProfileByName :one
SELECT * FROM transcode_profiles WHERE name = ? LIMIT 1;

-- name: CreateTranscodeProfile :one
INSERT INTO transcode_profiles(
  name,
  description,
  hls_segment_time,
  is_default,
  created_at
)
VALUES (?, ?, ?, ?, now())
RETURNING *;
