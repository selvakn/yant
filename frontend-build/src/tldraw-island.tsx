import { StrictMode, useEffect, useState, useCallback, useRef } from 'react'
import { createRoot } from 'react-dom/client'
import {
  Tldraw,
  createTLStore,
  defaultShapeUtils,
  defaultBindingUtils,
  loadSnapshot,
  getSnapshot,
  TLStoreSnapshot,
} from 'tldraw'
import 'tldraw/tldraw.css'

interface TldrawIslandProps {
  snapshotUrl: string
  saveUrl: string
  readOnly?: boolean
  container: HTMLElement
}

function TldrawIsland({ snapshotUrl, saveUrl, readOnly, container }: TldrawIslandProps) {
  const [store] = useState(() =>
    createTLStore({
      shapeUtils: defaultShapeUtils,
      bindingUtils: defaultBindingUtils,
    })
  )
  const [loaded, setLoaded] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
  const fadeTimerRef = useRef<ReturnType<typeof setTimeout>>()

  useEffect(() => {
    fetch(snapshotUrl, { credentials: 'same-origin' })
      .then((res) => {
        if (res.status === 404) return null
        if (!res.ok) throw new Error('Failed to load drawing')
        return res.json()
      })
      .then((data) => {
        if (data?.document) {
          loadSnapshot(store, data as TLStoreSnapshot)
        }
        setLoaded(true)
      })
      .catch((err) => {
        console.error('Error loading drawing:', err)
        setLoaded(true)
      })
  }, [snapshotUrl, store])

  // Auto-save on store changes
  useEffect(() => {
    if (readOnly || !loaded) return

    let saveTimeout: ReturnType<typeof setTimeout>
    let isMounted = true

    const unsub = store.listen(
      () => {
        clearTimeout(saveTimeout)
        saveTimeout = setTimeout(async () => {
          if (!isMounted) return
          setSaveStatus('saving')
          try {
            const snapshot = getSnapshot(store)
            const res = await fetch(saveUrl, {
              method: 'PUT',
              credentials: 'same-origin',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify(snapshot),
            })
            if (!res.ok) throw new Error('Save failed')
            if (!isMounted) return
            setSaveStatus('saved')
            clearTimeout(fadeTimerRef.current)
            fadeTimerRef.current = setTimeout(() => {
              if (isMounted) setSaveStatus('idle')
            }, 2500)
          } catch {
            if (isMounted) setSaveStatus('error')
          }
        }, 2000)
      },
      { source: 'user', scope: 'document' }
    )

    return () => {
      isMounted = false
      unsub()
      clearTimeout(saveTimeout)
      clearTimeout(fadeTimerRef.current)
    }
  }, [store, saveUrl, readOnly, loaded])

  const toggleFullscreen = useCallback(() => {
    if (!isFullscreen) {
      container.classList.add('tldraw-fullscreen')
      document.body.style.overflow = 'hidden'
    } else {
      container.classList.remove('tldraw-fullscreen')
      document.body.style.overflow = ''
    }
    setIsFullscreen(!isFullscreen)
  }, [isFullscreen, container])

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isFullscreen) {
        toggleFullscreen()
      }
    }
    document.addEventListener('keydown', handleEsc)
    return () => document.removeEventListener('keydown', handleEsc)
  }, [isFullscreen, toggleFullscreen])

  if (!loaded) {
    return <div className="tldraw-loading">Loading drawing...</div>
  }

  const statusText =
    saveStatus === 'saving' ? 'Saving\u2026' :
    saveStatus === 'saved' ? 'Saved' :
    saveStatus === 'error' ? 'Save failed' : ''

  const statusClass =
    saveStatus === 'saved' ? 'saved' :
    saveStatus === 'error' ? 'error' : ''

  return (
    <div className="tldraw-island-wrapper">
      <div className="tldraw-toolbar">
        {!readOnly && statusText && (
          <span className={`tldraw-save-status ${statusClass}`}>{statusText}</span>
        )}
        {!readOnly && !statusText && <span className="tldraw-save-status" />}
        <button onClick={toggleFullscreen} className="btn btn-secondary btn-sm">
          {isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
        </button>
      </div>
      <div className="tldraw-canvas-container">
        <Tldraw store={store} />
      </div>
    </div>
  )
}

declare global {
  interface Window {
    initTldrawIsland: (
      container: HTMLElement,
      snapshotUrl: string,
      saveUrl: string,
      options?: { readOnly?: boolean }
    ) => () => void
  }
}

window.initTldrawIsland = function (
  container: HTMLElement,
  snapshotUrl: string,
  saveUrl: string,
  options?: { readOnly?: boolean }
): () => void {
  const root = createRoot(container)
  root.render(
    <StrictMode>
      <TldrawIsland
        snapshotUrl={snapshotUrl}
        saveUrl={saveUrl}
        readOnly={options?.readOnly}
        container={container}
      />
    </StrictMode>
  )
  return () => {
    container.classList.remove('tldraw-fullscreen')
    document.body.style.overflow = ''
    root.unmount()
  }
}
