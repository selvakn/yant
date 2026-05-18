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
  DefaultColorThemePalette,
} from 'tldraw'
import 'tldraw/tldraw.css'

// Override tldraw's color palette with the app's tag color palette.
// Maps 10 tag colors onto 10 of tldraw's 13 named color slots;
// grey, light-violet, and white keep their defaults.
const tagColors: Array<{
  slot: keyof typeof DefaultColorThemePalette.lightMode
  solid: string
  semi: string
  pattern: string
  noteFill: string
}> = [
  { slot: 'black',       solid: '#001219', semi: '#c2d4d9', pattern: '#1a3540', noteFill: '#0a2a35' },
  { slot: 'blue',        solid: '#005f73', semi: '#b3dae3', pattern: '#1a7f93', noteFill: '#0a6f83' },
  { slot: 'light-blue',  solid: '#0a9396', semi: '#b5e2e3', pattern: '#2db0b3', noteFill: '#18a3a6' },
  { slot: 'green',       solid: '#94d2bd', semi: '#daf0e7', pattern: '#a8dcc9', noteFill: '#80c8ab' },
  { slot: 'yellow',      solid: '#e9d8a6', semi: '#f6f0db', pattern: '#efe0b8', noteFill: '#dfc882' },
  { slot: 'orange',      solid: '#ee9b00', semi: '#fae0b3', pattern: '#f2ad2e', noteFill: '#d48a00' },
  { slot: 'light-green', solid: '#ca6702', semi: '#f0d1a8', pattern: '#d9822e', noteFill: '#b45c02' },
  { slot: 'light-red',   solid: '#bb3e03', semi: '#ecc5b0', pattern: '#d05a28', noteFill: '#a53603' },
  { slot: 'red',         solid: '#ae2012', semi: '#e8bfba', pattern: '#c94536', noteFill: '#981c10' },
  { slot: 'violet',      solid: '#9b2226', semi: '#e3bfc0', pattern: '#b74448', noteFill: '#871e22' },
]

for (const c of tagColors) {
  const light = DefaultColorThemePalette.lightMode[c.slot]
  if (typeof light === 'object') {
    light.solid = c.solid
    light.fill = c.solid
    light.semi = c.semi
    light.pattern = c.pattern
    light.noteFill = c.noteFill
  }
  const dark = DefaultColorThemePalette.darkMode[c.slot]
  if (typeof dark === 'object') {
    dark.solid = c.solid
    dark.fill = c.solid
    dark.semi = c.semi
    dark.pattern = c.pattern
    dark.noteFill = c.noteFill
  }
}

DefaultColorThemePalette.lightMode.background = '#fefefe'
DefaultColorThemePalette.lightMode.solid = '#fefefe'
DefaultColorThemePalette.darkMode.background = '#fefefe'
DefaultColorThemePalette.darkMode.solid = '#fefefe'

interface TldrawIslandProps {
  snapshotUrl: string
  saveUrl: string
  readOnly?: boolean
  initialTool?: string
  licenseKey?: string
  container: HTMLElement
  onSvgReady?: (svg: string) => void
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

function TldrawIsland({ snapshotUrl, saveUrl, readOnly, initialTool, licenseKey, container, onSvgReady }: TldrawIslandProps) {
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

    let saveTimeout: ReturnType<typeof setTimeout> | null = null
    let isMounted = true
    let hasPendingChanges = false

    async function persist() {
      hasPendingChanges = false
      if (isMounted) {
        _saveStatus = 'saving'
        notifySaveStatus()
      }
      try {
        const snapshot = getSnapshot(store)
        const res = await fetch(saveUrl, {
          method: 'PUT',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(snapshot),
        })
        if (!res.ok) throw new Error('Save failed')
        if (isMounted) {
          _saveStatus = 'saved'
          notifySaveStatus()
        }

        const ed = editorRef.current
        if (ed) {
          const shapeIds = ed.getPageShapeIds(ed.getCurrentPage().id)
          if (shapeIds.size > 0) {
            try {
              const result = await ed.getSvgString([...shapeIds], { background: true, padding: 16 })
              if (result) {
                await fetch(saveUrl + '/svg', {
                  method: 'PUT',
                  credentials: 'same-origin',
                  headers: { 'Content-Type': 'image/svg+xml' },
                  body: result.svg,
                }).catch(() => {})
              }
            } catch { /* SVG export failed; JSON is saved at least */ }
          }
        }
        // Notify any external listeners that the disk SVG has been refreshed.
        try {
          const idMatch = /\/drawings\/([^/]+)$/.exec(saveUrl)
          window.dispatchEvent(new CustomEvent('yant:drawing-saved', {
            detail: { drawingID: idMatch ? idMatch[1] : null },
          }))
        } catch { /* no-op */ }
        if (isMounted) {
          clearTimeout(fadeTimerRef.current)
          fadeTimerRef.current = setTimeout(() => {
            if (isMounted) {
              _saveStatus = 'idle'
              notifySaveStatus()
            }
          }, 2500)
        }
      } catch {
        if (isMounted) {
          _saveStatus = 'error'
          notifySaveStatus()
        }
      }
    }

    const unsub = store.listen(
      () => {
        hasPendingChanges = true
        if (saveTimeout) clearTimeout(saveTimeout)
        saveTimeout = setTimeout(() => { persist() }, 2000)
      },
      { source: 'user', scope: 'document' }
    )

    return () => {
      if (saveTimeout) {
        clearTimeout(saveTimeout)
        saveTimeout = null
      }
      // Flush any pending edits before unmount so the disk SVG is current
      // by the time the host code refetches the preview.
      if (hasPendingChanges) {
        persist() // fire-and-forget; fetch outlives unmount
      }
      isMounted = false
      unsub()
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
          if (onSvgReady) {
            let exported = false
            const tryExport = async () => {
              if (exported) return
              try {
                const shapeIds = editor.getPageShapeIds(editor.getCurrentPage().id)
                console.log('[tldraw-export] tryExport shapeIds.size:', shapeIds.size)
                if (shapeIds.size === 0) return
                const result = await editor.getSvgString([...shapeIds], { background: true, padding: 16 })
                console.log('[tldraw-export] getSvgString result:', result ? 'ok svg.length=' + result.svg?.length : 'null/undefined')
                if (result?.svg) {
                  exported = true
                  onSvgReady(result.svg)
                }
              } catch (e) {
                console.error('[tldraw-export] SVG export failed:', e)
              }
            }
            // Retry at increasing intervals to handle font loading and layout settling
            setTimeout(tryExport, 500)
            setTimeout(tryExport, 1500)
            setTimeout(tryExport, 4000)
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
      options?: { readOnly?: boolean; initialTool?: string; licenseKey?: string; onSvgReady?: (svg: string) => void }
    ) => () => void
  }
}

window.initTldrawIsland = function (
  container: HTMLElement,
  snapshotUrl: string,
  saveUrl: string,
  options?: { readOnly?: boolean; initialTool?: string; licenseKey?: string; onSvgReady?: (svg: string) => void }
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
        onSvgReady={options?.onSvgReady}
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
