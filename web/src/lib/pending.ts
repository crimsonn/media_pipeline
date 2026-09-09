import { api } from '#/lib/api'

/** Mirrors domain.PendingFileResponse (internal/domain/pending.go). */
export interface PendingFile {
  id: number
  file_name: string
  status: string
  created_at: string
  updated_at: string
}

/**
 * Mirrors domain.PendingEnqueueRequest. Both fields use `binding:"required"`
 * on the Go side, so `transcode_profile_id` must be > 0.
 */
export interface PendingEnqueueInput {
  id: number
  file_name: string
  transcode_profile_id: number
}

/**
 * GET /api/v1/pending/files
 * Files dropped into the watch folder that are waiting for a transcode
 * profile to be assigned.
 */
export async function listPendingFiles(): Promise<PendingFile[]> {
  const { data } = await api.get<PendingFile[]>('/pending/files')
  return data ?? []
}

/**
 * POST /api/v1/pending/files
 * Assigns a transcode profile to a pending file and enqueues it for transcoding.
 */
export async function enqueuePendingFile(
  input: PendingEnqueueInput,
): Promise<void> {
  await api.post('/pending/files', input)
}
