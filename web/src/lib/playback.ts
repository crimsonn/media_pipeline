import { api, baseURL } from '#/lib/api'

/** Mirrors domain.PlaybackOutput (internal/domain/playback.go). */
export interface PlaybackOutput {
  name: string
  /** Master playlist path relative to the API v1 base, e.g. "playback/hls/clip/master.m3u8". */
  master_path: string
  renditions: string[]
  modified_at: string
}

/** GET /api/v1/playback/outputs — finished HLS transcodes in the output folder, newest first. */
export async function listOutputs(): Promise<PlaybackOutput[]> {
  const { data } = await api.get<PlaybackOutput[]>('/playback/outputs')
  return data ?? []
}

/** Absolute URL to a playlist/segment served by the API, given an API-relative path. */
export function playbackUrl(relativePath: string): string {
  return `${baseURL.replace(/\/$/, '')}/${relativePath.replace(/^\//, '')}`
}
