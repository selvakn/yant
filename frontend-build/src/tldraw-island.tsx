import { StrictMode, useEffect, useState, useCallback, useRef } from 'react'
import { createRoot } from 'react-dom/client'
import {
  Tldraw,
  DefaultActionsMenu,
  DefaultActionsMenuContent,
  TldrawUiMenuItem,
  TLComponents,
  createTLStore,
  defaultShapeUtils,
  defaultBindingUtils,
  loadSnapshot,
  getSnapshot,
} from 'tldraw'
import 'tldraw/tldraw.css'

interface TldrawIslandProps {
  snapshotUrl: string
  saveUrl: string
  readOnly?: boolean
  initialTool?: string
  licenseKey?: string
  container: HTMLElement
}

// Shared state between tldraw components and the island wrapper
let _isFullscreen = false
let _toggleFullscreen: (() => void) | null = null
let _saveStatus: 'idle' | 'saving' | 'saved' | 'error' = 'idle'
let _saveStatusListeners: Array<() => void> = []

function notifySaveStatus() {
  _saveStatusListeners.forEach((fn) => fn())
}

function CustomActionsMenu() {
  const [fs, setFs] = useState(_isFullscreen)

  useEffect(() => {
    const interval = setInterval(() => {
      if (fs !== _isFullscreen) setFs(_isFullscreen)
    }, 100)
    return () => clearInterval(interval)
  }, [fs])

  return (
    <DefaultActionsMenu>
      <TldrawUiMenuItem
        id="toggle-fullscreen"
        label={fs ? 'Exit fullscreen' : 'Fullscreen'}
        icon="external-link"
        readonlyOk
        kbd="shift+f"
        onSelect={() => {
          if (_toggleFullscreen) _toggleFullscreen()
          setFs(!fs)
        }}
      />
      <DefaultActionsMenuContent />
    </DefaultActionsMenu>
  )
}

function SaveStatusIndicator() {
  const [status, setStatus] = useState(_saveStatus)

  useEffect(() => {
    const listener = () => setStatus(_saveStatus)
    _saveStatusListeners.push(listener)
    return () => {
      _saveStatusListeners = _saveStatusListeners.filter((l) => l !== listener)
    }
  }, [])

  if (status === 'idle') return null

  const text =
    status === 'saving' ? 'Saving\u2026' :
    status === 'saved' ? 'Saved' :
    status === 'error' ? 'Save failed' : ''

  const className =
    status === 'saved' ? 'tldraw-save-indicator saved' :
    status === 'error' ? 'tldraw-save-indicator error' :
    'tldraw-save-indicator'

  return <div className={className}>{text}</div>
}

function TldrawIsland({ snapshotUrl, saveUrl, readOnly, initialTool, licenseKey, container }: TldrawIslandProps) {
  const [store] = useState(() =>
    createTLStore({
      shapeUtils: defaultShapeUtils,
      bindingUtils: defaultBindingUtils,
    })
  )
  const [loaded, setLoaded] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const fadeTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const editorRef = useRef<any>(null)

  useEffect(() => {
    fetch(snapshotUrl, { credentials: 'same-origin' })
      .then((res) => {
        if (res.status === 404) return null
        if (!res.ok) throw new Error('Failed to load drawing')
        return res.json()
      })
      .then((data) => {
        if (data?.type === 'tldraw' && data?.document) {
          loadSnapshot(store, data.document)
        } else if (data?.document) {
          loadSnapshot(store, data)
        }
        setLoaded(true)
      })
      .catch((err) => {
        console.error('Error loading drawing:', err)
        setLoaded(true)
      })
  }, [snapshotUrl, store])

  useEffect(() => {
    if (readOnly || !loaded) return

    let saveTimeout: ReturnType<typeof setTimeout>
    let isMounted = true

    const unsub = store.listen(
      () => {
        clearTimeout(saveTimeout)
        saveTimeout = setTimeout(async () => {
          if (!isMounted) return
          _saveStatus = 'saving'
          notifySaveStatus()
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
            _saveStatus = 'saved'
            notifySaveStatus()

            const ed = editorRef.current
            if (ed) {
              const shapeIds = ed.getPageShapeIds(ed.getCurrentPage().id)
              if (shapeIds.size > 0) {
                ed.getSvgString([...shapeIds], { background: true, padding: 16 })
                  .then((result: { svg: string } | null) => {
                    if (!result) return
                    fetch(saveUrl + '/svg', {
                      method: 'PUT',
                      credentials: 'same-origin',
                      headers: { 'Content-Type': 'image/svg+xml' },
                      body: result.svg,
                    }).catch(() => {})
                  })
                  .catch(() => {})
              }
            }
            clearTimeout(fadeTimerRef.current)
            fadeTimerRef.current = setTimeout(() => {
              if (isMounted) {
                _saveStatus = 'idle'
                notifySaveStatus()
              }
            }, 2500)
          } catch {
            if (isMounted) {
              _saveStatus = 'error'
              notifySaveStatus()
            }
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
    if (!document.fullscreenElement) {
      container.requestFullscreen().catch(() => {})
    } else {
      document.exitFullscreen().catch(() => {})
    }
  }, [container])

  useEffect(() => {
    const onFsChange = () => {
      const fs = document.fullscreenElement === container
      _isFullscreen = fs
      setIsFullscreen(fs)
    }
    document.addEventListener('fullscreenchange', onFsChange)
    return () => document.removeEventListener('fullscreenchange', onFsChange)
  }, [container])

  // Keep the shared ref in sync
  useEffect(() => {
    _toggleFullscreen = toggleFullscreen
    return () => { _toggleFullscreen = null }
  }, [toggleFullscreen])

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'F' && e.shiftKey && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const tag = (e.target as HTMLElement)?.tagName
        if (tag !== 'INPUT' && tag !== 'TEXTAREA') {
          toggleFullscreen()
        }
      }
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [isFullscreen, toggleFullscreen])

  if (!loaded) {
    return <div className="tldraw-loading">Loading drawing...</div>
  }

  const components: TLComponents = {
    ActionsMenu: CustomActionsMenu,
    HelpMenu: null,
    DebugMenu: null,
    PageMenu: null,
    SharePanel: readOnly ? null : SaveStatusIndicator,
  }

  return (
    <div className="tldraw-canvas-container" style={{ height: '100%' }}>
      <Tldraw
        store={store}
        licenseKey={licenseKey || undefined}
        initialState={initialTool || 'select'}
        components={components}
        onMount={(editor) => {
          editorRef.current = editor
          if (readOnly) {
            editor.updateInstanceState({ isReadonly: true })
          }
        }}
      />
    </div>
  )
}

declare global {
  interface Window {
    initTldrawIsland: (
      container: HTMLElement,
      snapshotUrl: string,
      saveUrl: string,
      options?: { readOnly?: boolean; initialTool?: string; licenseKey?: string }
    ) => () => void
  }
}

window.initTldrawIsland = function (
  container: HTMLElement,
  snapshotUrl: string,
  saveUrl: string,
  options?: { readOnly?: boolean; initialTool?: string; licenseKey?: string }
): () => void {
  _isFullscreen = false
  _saveStatus = 'idle'
  _saveStatusListeners = []

  const root = createRoot(container)
  root.render(
    <StrictMode>
      <TldrawIsland
        snapshotUrl={snapshotUrl}
        saveUrl={saveUrl}
        readOnly={options?.readOnly}
        initialTool={options?.initialTool}
        licenseKey={options?.licenseKey}
        container={container}
      />
    </StrictMode>
  )
  return () => {
    if (document.fullscreenElement === container) {
      document.exitFullscreen().catch(() => {})
    }
    root.unmount()
  }
}
