import { getUnsyncedProgress, markSynced } from '../db/idb'

const API_BASE = '/api'

function getToken(): string | null {
  const remember = localStorage.getItem('remember_me') === 'true'
  const store = remember ? localStorage : sessionStorage
  return store.getItem('access_token')
}

async function drainQueue(): Promise<void> {
  const entries = await getUnsyncedProgress()
  if (entries.length === 0) return

  const token = getToken()
  if (!token) return

  for (const entry of entries) {
    try {
      const res = await fetch(`${API_BASE}/progress/${entry.courseId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          lesson_seq: entry.lessonSeq,
          completed: entry.completed,
        }),
      })
      if (res.ok) {
        await markSynced(entry.courseId, entry.lessonSeq)
      }
    } catch {
      // Will retry on next online event
    }
  }
}

export function syncNow(): Promise<void> {
  return drainQueue()
}

export function startSyncManager(): () => void {
  const handler = () => {
    drainQueue().catch(console.error)
  }

  // Sync immediately if already online
  if (navigator.onLine) {
    drainQueue().catch(console.error)
  }

  window.addEventListener('online', handler)
  return () => window.removeEventListener('online', handler)
}
