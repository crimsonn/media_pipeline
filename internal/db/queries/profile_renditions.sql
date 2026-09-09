-- name: LinkProfileRendition :exec
INSERT INTO profile_renditions (profile_id, rendition_id, stream_index)
VALUES ($1, $2, $3);
