PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS renditions(
  id integer PRIMARY KEY,
  name varchar(50) NOT NULL UNIQUE,
  width int NOT NULL,
  height int NOT NULL,
  video_bitrate int NOT NULL,
  audio_bitrate int NOT NULL,
  video_codec varchar(30) DEFAULT 'h265',
  audio_codec varchar(30) DEFAULT 'aac',
  fps int DEFAULT 30,
  created_at timestamp with time zone DEFAULT current_timestamp
);

CREATE TABLE IF NOT EXISTS transcode_profiles(
  id integer PRIMARY KEY,
  name varchar(50) NOT NULL UNIQUE,
  description text,
  hls_segment_time int DEFAULT 6,
  is_default boolean DEFAULT false,
  created_at timestamp with time zone DEFAULT current_timestamp
);

CREATE TABLE IF NOT EXISTS profile_renditions(
  profile_id int REFERENCES transcode_profiles(id) ON DELETE CASCADE,
  rendition_id int REFERENCES renditions(id) ON DELETE CASCADE,
  stream_index int NOT NULL,
  PRIMARY KEY(profile_id, rendition_id)
);
