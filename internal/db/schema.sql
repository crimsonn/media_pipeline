CREATE TABLE IF NOT EXISTS renditions (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE,
  width INT NOT NULL,
  height INT NOT NULL,
  video_bitrate INT NOT NULL,
  audio_bitrate INT NOT NULL,
  video_codec VARCHAR(30) NOT NULL DEFAULT 'h264',
  audio_codec VARCHAR(30) NOT NULL DEFAULT 'aac',
  fps INT NOT NULL DEFAULT 30,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transcode_profiles (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE,
  description TEXT,
  hls_segment_time INT NOT NULL DEFAULT 6,
  is_default BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS profile_renditions (
  profile_id BIGINT NOT NULL REFERENCES transcode_profiles(id) ON DELETE CASCADE,
  rendition_id BIGINT NOT NULL REFERENCES renditions(id) ON DELETE CASCADE,
  stream_index INT NOT NULL,
  PRIMARY KEY (profile_id, rendition_id)
);

CREATE TABLE IF NOT EXISTS jobs (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  source_path TEXT NOT NULL,
  output_dir TEXT NOT NULL,
  file_name TEXT NOT NULL DEFAULT '',
  file_id TEXT,
  profile_id BIGINT NOT NULL REFERENCES transcode_profiles(id),
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'running', 'completed', 'failed')),
  progress_percent REAL NOT NULL DEFAULT 0,
  error_message TEXT,
  retry_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS jobs_pending_created_at_idx
  ON jobs (created_at)
  WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS job_tasks (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  rendition_id BIGINT NOT NULL REFERENCES renditions(id),
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'running', 'completed', 'failed')),
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (job_id, rendition_id)
);

CREATE INDEX IF NOT EXISTS job_tasks_pending_created_at_idx
  ON job_tasks (created_at)
  WHERE status = 'pending';
