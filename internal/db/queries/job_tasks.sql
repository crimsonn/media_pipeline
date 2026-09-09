-- name: InsertJobTask :one
INSERT INTO job_tasks (job_id, rendition_id)
VALUES ($1, $2)
RETURNING *;

-- name: ClaimJobTask :one
WITH next_task AS (
  SELECT id
  FROM job_tasks
  WHERE status = 'pending'
  ORDER BY created_at
  LIMIT 1
  FOR UPDATE SKIP LOCKED
), claimed AS (
  UPDATE job_tasks t
  SET
    status = 'running',
    updated_at = now()
  FROM next_task
  WHERE t.id = next_task.id
  RETURNING t.*
)
SELECT
  c.id AS task_id,
  c.job_id,
  c.rendition_id,
  j.source_path,
  j.output_dir,
  j.file_name,
  p.hls_segment_time,
  r.name AS rendition_name,
  r.width,
  r.height,
  r.video_bitrate,
  r.audio_bitrate,
  r.video_codec,
  r.audio_codec
FROM claimed c
JOIN jobs j ON j.id = c.job_id
JOIN renditions r ON r.id = c.rendition_id
JOIN transcode_profiles p ON p.id = j.profile_id;

-- name: MarkJobRunning :exec
UPDATE jobs
SET
  status = 'running',
  updated_at = now()
WHERE id = $1 AND status = 'pending';

-- name: CompleteJobTask :exec
UPDATE job_tasks
SET
  status = 'completed',
  error_message = NULL,
  updated_at = now()
WHERE id = $1;

-- name: FailJobTask :exec
UPDATE job_tasks
SET
  status = 'failed',
  error_message = $2,
  updated_at = now()
WHERE id = $1;

-- name: JobTaskStats :one
SELECT
  COUNT(*) FILTER (WHERE status = 'pending')::bigint AS pending,
  COUNT(*) FILTER (WHERE status = 'running')::bigint AS running,
  COUNT(*) FILTER (WHERE status = 'completed')::bigint AS completed,
  COUNT(*) FILTER (WHERE status = 'failed')::bigint AS failed
FROM job_tasks
WHERE job_id = $1;

-- name: ListJobRenditions :many
SELECT
  r.id,
  r.name,
  r.width,
  r.height,
  r.video_bitrate,
  r.audio_bitrate,
  r.video_codec,
  r.audio_codec,
  pr.stream_index
FROM job_tasks jt
JOIN jobs j ON j.id = jt.job_id
JOIN renditions r ON r.id = jt.rendition_id
LEFT JOIN profile_renditions pr
  ON pr.profile_id = j.profile_id AND pr.rendition_id = r.id
WHERE jt.job_id = $1
ORDER BY pr.stream_index NULLS LAST, r.name;


-- name: GetLatestJobTasks :many
WITH latest_jobs AS (
  SELECT
    id,
    status,
    error_message,
    created_at,
    updated_at,
    profile_id,
    source_path,
    output_dir,
    file_name
  FROM jobs
  ORDER BY created_at DESC
  LIMIT 10
)
SELECT
  j.id AS job_id,
  j.status AS job_status,
  j.error_message AS job_error_message,
  j.created_at AS job_created_at,
  j.updated_at AS job_updated_at,
  j.profile_id AS job_profile_id,
  j.source_path AS job_source_path,
  j.output_dir AS job_output_dir,
  j.file_name AS job_file_name,
  jt.id AS task_id,
  jt.status AS task_status,
  jt.created_at AS task_created_at,
  jt.updated_at AS task_updated_at,
  jt.error_message AS task_error_message,
  r.id AS rendition_id,
  r.name AS rendition_name,
  r.width AS rendition_width,
  r.height AS rendition_height,
  r.video_bitrate AS rendition_video_bitrate,
  r.audio_bitrate AS rendition_audio_bitrate,
  r.video_codec AS rendition_video_codec,
  r.audio_codec AS rendition_audio_codec,
  r.fps AS rendition_fps
FROM latest_jobs AS j
INNER JOIN job_tasks AS jt ON jt.job_id = j.id
INNER JOIN renditions AS r ON r.id = jt.rendition_id
ORDER BY j.created_at DESC, jt.id;