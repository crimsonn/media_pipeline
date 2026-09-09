import axios from 'axios'

/**
 * Base URL of the transcoder REST API (Gin server, see internal/server/router.go).
 * Override in web/.env with VITE_API_URL, e.g. http://localhost:8080/api/v1
 */
export const baseURL =
  import.meta.env.VITE_API_URL ?? 'http://localhost:8080/api/v1'

export const api = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' },
})

/** Pull a human-readable message out of an axios error, falling back sensibly. */
export function apiErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as { error?: string } | undefined
    return data?.error ?? error.message
  }
  return error instanceof Error ? error.message : 'Unexpected error'
}
