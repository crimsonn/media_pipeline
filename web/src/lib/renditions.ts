import { api } from '#/lib/api'

/** Mirrors domain.RenditionResponse (internal/domain/transcoder.go). */
export interface Rendition {
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

/**
 * Mirrors domain.CreateRenditionRequest. Every field is required and the Go
 * side uses `binding:"required"`, so numbers must be > 0.
 */
export type CreateRenditionInput = Omit<Rendition, 'id'>

/** GET /api/v1/transcoder/renditions */
export async function listRenditions(): Promise<Rendition[]> {
  const { data } = await api.get<Rendition[]>('/transcoder/renditions')
  return data ?? []
}

/** POST /api/v1/transcoder/renditions */
export async function createRendition(
  input: CreateRenditionInput,
): Promise<Rendition> {
  const { data } = await api.post<Rendition>('/transcoder/renditions', input)
  return data
}

/** DELETE /api/v1/transcoder/renditions/:id */
export async function deleteRendition(id: number): Promise<void> {
  await api.delete(`/transcoder/renditions/${id}`)
}
