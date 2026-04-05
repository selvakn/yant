import { StrictMode, useEffect, useState, useCallback } from 'react'
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
  onClose?: () => void
  readOnly?: boolean
  container: HTMLElement
}

function TldrawIsland({ snapshotUrl, saveUrl, onClose, readOnly, container }: TldrawIslandProps) {
  const [store] = useState(() =>
    createTLStore({
      shapeUtils: defaultShapeUtils,
      bindingUtils: defaultBindingUtils,
    })
  )
  const [loaded, setLoaded] = useState(false)
  const [saving, setSaving] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)

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

  const handleSave = async () => {
    setSaving(true)
    try {
      const snapshot = getSnapshot(store)
      const res = await fetch(saveUrl, {
        method: 'PUT',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(snapshot),
      })
      if (!res.ok) throw new Error('Save failed')
    } catch (err) {
      console.error('Error saving drawing:', err)
      alert('Failed to save drawing')
    } finally {
      setSaving(false)
    }
  }

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

  return (
    <div className="tldraw-island-wrapper">
      <div className="tldraw-toolbar">
        {!readOnly && (
          <button onClick={handleSave} disabled={saving} className="btn btn-primary">
            {saving ? 'Saving...' : 'Save Drawing'}
          </button>
        )}
        <button onClick={toggleFullscreen} className="btn btn-secondary">
          {isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
        </button>
        {onClose && !isFullscreen && (
          <button onClick={onClose} className="btn btn-secondary">
            Close
          </button>
        )}
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
      options?: { onClose?: () => void; readOnly?: boolean }
    ) => () => void
  }
}

window.initTldrawIsland = function (
  container: HTMLElement,
  snapshotUrl: string,
  saveUrl: string,
  options?: { onClose?: () => void; readOnly?: boolean }
): () => void {
  const root = createRoot(container)
  root.render(
    <StrictMode>
      <TldrawIsland
        snapshotUrl={snapshotUrl}
        saveUrl={saveUrl}
        onClose={options?.onClose}
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
