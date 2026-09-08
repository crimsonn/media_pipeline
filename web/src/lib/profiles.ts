import { api } from '#/lib/api'
import type { Rendition } from '#/lib/renditions'

/**
 * Renditions come back slightly differently depending on the endpoint:
 * POST /profiles returns domain.RenditionResponse (has `fps`), while
 * GET /profiles/:id returns domain.Rendition (has `stream_index`).
 */
export interface ProfileRendition extends Partial<Rendition> {
  id: number
  name: string
  width: number
  height: number
  video_bitrate: number
  audio_bitrate: number
  video_codec: string
  audio_codec: string
  fps?: number
  stream_index?: number
}

/** Mirrors domain.Profile / domain.ProfileResponse. */
export interface Profile {
  id: number
  name: string
  description: string
  hls_segment_time: number
  renditions: ProfileRendition[]
}

/**
 * Mirrors domain.CreateProfileRequest. Every field uses `binding:"required"`,
 * so `hls_segment_time` must be > 0, `renditions` must be non-empty, and
 * `is_default` must be `true` (a false bool fails gin's "required" tag).
 */
export interface CreateProfileInput {
  name: string
  description: string
  hls_segment_time: number
  is_default: boolean
  /** Rendition IDs; array order becomes each rendition's stream_index. */
  renditions: number[]
}

/** GET /api/v1/transcoder/profiles */
export async function listProfiles(): Promise<Profile[]> {
  const { data } = await api.get<Profile[]>('/transcoder/profiles')
  return data ?? []
}

/** POST /api/v1/transcoder/profiles */
export async function createProfile(
  input: CreateProfileInput,
): Promise<Profile> {
  const { data } = await api.post<Profile>('/transcoder/profiles', input)
  return data
}

/** GET /api/v1/transcoder/profiles/:id */
export async function getProfile(id: number): Promise<Profile> {
  const { data } = await api.get<Profile>(`/transcoder/profiles/${id}`)
  return data
}
