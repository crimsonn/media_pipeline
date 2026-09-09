import { useEffect, useRef, useState } from 'react'
import type Hls from 'hls.js'

interface HlsPlayerProps {
  /** Absolute URL of the master (or variant) .m3u8 playlist. */
  src: string
  className?: string
}

/**
 * <video> element wired to hls.js. Safari/iOS play HLS natively, so hls.js is
 * only loaded (dynamically, to keep it out of the SSR bundle) when the browser
 * needs it.
 */
export function HlsPlayer({ src, className }: HlsPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    setError(null)

    // Native HLS (Safari, iOS, some smart TVs).
    if (video.canPlayType('application/vnd.apple.mpegurl')) {
      video.src = src
      return () => {
        video.removeAttribute('src')
        video.load()
      }
    }

    let cancelled = false

    import('hls.js')
      .then(({ default: HlsCtor }) => {
        if (cancelled || !videoRef.current) return

        if (!HlsCtor.isSupported()) {
          setError('HLS playback is not supported in this browser.')
          return
        }

        const hls = new HlsCtor({ enableWorker: true })
        hlsRef.current = hls
        hls.loadSource(src)
        hls.attachMedia(videoRef.current)

        hls.on(HlsCtor.Events.ERROR, (_event, data) => {
          if (!data.fatal) return
          if (data.type === HlsCtor.ErrorTypes.NETWORK_ERROR) {
            hls.startLoad()
          } else if (data.type === HlsCtor.ErrorTypes.MEDIA_ERROR) {
            hls.recoverMediaError()
          } else {
            setError(`Playback error: ${data.details}`)
            hls.destroy()
            hlsRef.current = null
          }
        })
      })
      .catch(() => {
        if (!cancelled) setError('Failed to load the video player.')
      })

    return () => {
      cancelled = true
      hlsRef.current?.destroy()
      hlsRef.current = null
    }
  }, [src])

  return (
    <div className={className}>
      <video
        ref={videoRef}
        controls
        playsInline
        className="aspect-video w-full rounded-lg border border-border bg-black"
      >
        <track kind="captions" />
      </video>
      {error ? (
        <p className="mt-2 text-xs text-destructive">{error}</p>
      ) : null}
    </div>
  )
}
