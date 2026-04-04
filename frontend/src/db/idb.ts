const DB_NAME = 'lang-learn'
const DB_VERSION = 1

interface ProgressEntry {
  courseId: string
  lessonSeq: number
  completed: boolean
  timestamp: number
  synced: boolean
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION)
    req.onupgradeneeded = () => {
      const db = req.result
      if (!db.objectStoreNames.contains('progress')) {
        const store = db.createObjectStore('progress', { keyPath: ['courseId', 'lessonSeq'] })
        store.createIndex('synced', 'synced', { unique: false })
      }
      if (!db.objectStoreNames.contains('courses')) {
        db.createObjectStore('courses', { keyPath: 'id' })
      }
      if (!db.objectStoreNames.contains('audio')) {
        db.createObjectStore('audio', { keyPath: 'key' })
      }
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

export async function saveProgress(entry: ProgressEntry): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction('progress', 'readwrite')
    tx.objectStore('progress').put(entry)
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

export async function getProgress(courseId: string): Promise<ProgressEntry[]> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction('progress', 'readonly')
    const store = tx.objectStore('progress')
    const entries: ProgressEntry[] = []
    const req = store.openCursor()
    req.onsuccess = () => {
      const cursor = req.result
      if (cursor) {
        const val = cursor.value as ProgressEntry
        if (val.courseId === courseId) entries.push(val)
        cursor.continue()
      } else {
        resolve(entries)
      }
    }
    req.onerror = () => reject(req.error)
  })
}

export async function getUnsyncedProgress(): Promise<ProgressEntry[]> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction('progress', 'readonly')
    const index = tx.objectStore('progress').index('synced')
    const req = index.getAll(IDBKeyRange.only(false))
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

export async function markSynced(courseId: string, lessonSeq: number): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction('progress', 'readwrite')
    const store = tx.objectStore('progress')
    const getReq = store.get([courseId, lessonSeq])
    getReq.onsuccess = () => {
      if (getReq.result) {
        store.put({ ...getReq.result, synced: true })
      }
      tx.oncomplete = () => resolve()
    }
    tx.onerror = () => reject(tx.error)
  })
}

export async function saveCourseOffline(course: unknown): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction('courses', 'readwrite')
    tx.objectStore('courses').put(course)
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

export async function getOfflineCourse(id: string): Promise<unknown | undefined> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction('courses', 'readonly')
    const req = tx.objectStore('courses').get(id)
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

export async function saveAudioOffline(key: string, data: ArrayBuffer): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction('audio', 'readwrite')
    tx.objectStore('audio').put({ key, data })
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

export async function getOfflineAudio(key: string): Promise<ArrayBuffer | undefined> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction('audio', 'readonly')
    const req = tx.objectStore('audio').get(key)
    req.onsuccess = () => resolve(req.result?.data)
    req.onerror = () => reject(req.error)
  })
}
