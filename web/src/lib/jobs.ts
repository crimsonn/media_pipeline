import { api } from '#/lib/api'

export type JobStatus = 'pending' | 'running' | 'completed' | 'failed'

/** Mirrors domain.RenditionResponse (internal/domain/transcoder.go). */
export interface JobRendition {
  id: number
  name: string
  width: number
  height: number
  video_bitrate: number
  audio_bitrate: number
  video_codec: string
  audio_codec: string
  fps: number
}

/** Mirrors domain.JobTaskResponse — one rendition being encoded for a job. */
export interface JobTask {
  id: number
  status: JobStatus
  created_at: string
  updated_at: string
  error_message: string
  rendition: JobRendition
}

/** Mirrors domain.JobResponse (internal/domain/jobs.go). */
export interface Job {
  id: number
  status: JobStatus
  error_message: string
  created_at: string
  updated_at: string
  profile_id: number
  source_path: string
  output_dir: string
  file_name: string
  tasks: JobTask[]
}

/**
 * GET /api/v1/jobs/latest
 * The 10 most recent jobs (newest first), each with its per-rendition tasks.
 * The API does not expose a numeric progress field, so callers derive progress
 * from the task statuses.
 */
export async function listLatestJobs(): Promise<Job[]> {
  const { data } = await api.get<Job[]>('/jobs/latest')
  return data ?? []
}

/** Fraction (0–1) of a job's rendition tasks that have finished (completed or failed). */
export function jobProgress(job: Job): number {
  if (job.tasks.length === 0) return job.status === 'completed' ? 1 : 0
  const done = job.tasks.filter(
    (t) => t.status === 'completed' || t.status === 'failed',
  ).length
  return done / job.tasks.length
}
