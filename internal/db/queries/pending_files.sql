-- name: CreatePendingFile :one
INSERT INTO pending_files (file_name) VALUES ($1) RETURNING *;

-- name: GetPendingFiles :many
SELECT * FROM pending_files WHERE status = 'waiting';

-- name: GetPendingFile :one
SELECT * FROM pending_files WHERE file_name = $1 AND status = 'waiting';

-- name: UpdatePendingFileStatus :exec
UPDATE pending_files SET status = $2 WHERE id = $1;