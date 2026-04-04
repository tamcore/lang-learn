import { useState } from 'react'
import { saveCourseOffline } from '../../db/idb'
import type { Lesson, Turn } from '../../api/types'

const AUDIO_CACHE = 'audio-cache'

interface Props {
  courseId: string
  lesson: Lesson
}

export default function DownloadOffline({ courseId, lesson }: Props) {
  const [progress, setProgress] = useState<number | null>(null)
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const audioTurns = lesson.turns.filter(
    (t: Turn) => t.speaker === 'system' && t.audio_file
  )

  async function handleDownload() {
    if (audioTurns.length === 0) {
      setDone(true)
      await saveCourseOffline({ id: courseId, lessonId: lesson.id, availableOffline: true })
      return
    }

    setProgress(0)
    setError(null)
    setDone(false)

    try {
      const cache = await caches.open(AUDIO_CACHE)
      let completed = 0

      for (const turn of audioTurns) {
        const url = `/api/audio/${turn.audio_file}`
        try {
          await cache.add(url)
        } catch {
          // Retry once
          try {
            await cache.add(url)
          } catch {
            // Skip this file on second failure
          }
        }
        completed++
        setProgress(Math.round((completed / audioTurns.length) * 100))
      }

      await saveCourseOffline({
        id: courseId,
        lessonId: lesson.id,
        availableOffline: true,
      })
      setDone(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Download failed')
    } finally {
      if (!done) setProgress(null)
    }
  }

  if (done) {
    return (
      <button className="btn-primary" disabled style={{ opacity: 0.7, padding: '0.3rem 0.8rem', fontSize: '0.8rem' }}>
        ✓ Available Offline
      </button>
    )
  }

  if (progress !== null) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
        <div style={{
          width: '80px',
          height: '6px',
          background: 'var(--border)',
          borderRadius: '3px',
          overflow: 'hidden',
        }}>
          <div style={{
            width: `${progress}%`,
            height: '100%',
            background: 'var(--primary)',
            borderRadius: '3px',
            transition: 'width 0.3s',
          }} />
        </div>
        <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>{progress}%</span>
      </div>
    )
  }

  return (
    <>
      <button
        className="btn-primary"
        onClick={handleDownload}
        style={{ padding: '0.3rem 0.8rem', fontSize: '0.8rem' }}
      >
        ⬇ Download for Offline
      </button>
      {error && <span className="error" style={{ marginLeft: '0.5rem' }}>{error}</span>}
    </>
  )
}
