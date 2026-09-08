-- name: EnqueueJob :one
INSERT INTO jobs (
  source_path,
  output_dir,
  file_name,
  file_id,
  profile_id
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = $1 LIMIT 1;

-- name: ClaimJob :one
WITH next_job AS (
  SELECT id
  FROM jobs
  WHERE status = 'pending'
  ORDER BY created_at
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
UPDATE jobs
SET
  status = 'running',
  updated_at = now()
FROM next_job
WHERE jobs.id = next_job.id
RETURNING jobs.*;

-- name: UpdateJobProgress :exec
UPDATE jobs
SET
  progress_percent = $2,
  updated_at = now()
WHERE id = $1;

-- name: CompleteJob :exec
UPDATE jobs
SET
  status = 'completed',
  progress_percent = 100,
  error_message = NULL,
  updated_at = now()
WHERE id = $1;

-- name: FailJob :exec
UPDATE jobs
SET
  status = 'failed',
  error_message = $2,
  retry_count = retry_count + 1,
  updated_at = now()
WHERE id = $1;
