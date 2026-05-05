import { StrictMode, useEffect, useState, useCallback, useRef } from 'react'
import { createRoot } from 'react-dom/client'
import { Excalidraw, serializeAsJSON, restore, exportToSvg } from '@excalidraw/excalidraw'
import '@excalidraw/excalidraw/index.css'

// Firefox compat: ensure FontFace.unicodeRange never returns undefined.
// Excalidraw 0.18.x exportToSvg crashes in Firefox because
// getUnicodeRangeRegex() calls .split() on undefined unicodeRange.
// See https://github.com/excalidraw/excalidraw/issues/10604
;(() => {
  if (typeof FontFace === 'undefined') return
  try {
    const desc = Object.getOwnPropertyDescriptor(FontFace.prototype, 'unicodeRange')
    if (!desc || !desc.get) return
    const origGet = desc.get
    Object.defineProperty(FontFace.prototype, 'unicodeRange', {
      get() {
        try {
          const val = origGet.call(this)
          return val ?? 'U+0-10FFFF'
        } catch {
          return 'U+0-10FFFF'
        }
      },
      set: desc.set,
      configurable: true,
      enumerable: desc.enumerable,
    })
  } catch { /* patch failed — exportToSvg may fail in Firefox */ }
})()

interface ExcalidrawIslandProps {
  snapshotUrl: string
  saveUrl: string
  readOnly?: boolean
  container: HTMLElement
}

let _saveStatus: 'idle' | 'saving' | 'saved' | 'error' = 'idle'
let _saveStatusListeners: Array<() => void> = []

function notifySaveStatus() {
  _saveStatusListeners.forEach((fn) => fn())
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
    status === 'saved' ? 'excalidraw-save-indicator saved' :
    status === 'error' ? 'excalidraw-save-indicator error' :
    'excalidraw-save-indicator'

  return <div className={className}>{text}</div>
}

function ExcalidrawIsland({ snapshotUrl, saveUrl, readOnly, container }: ExcalidrawIslandProps) {
  const [initialData, setInitialData] = useState<any>(null)
  const [loaded, setLoaded] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const apiRef = useRef<any>(null)
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const fadeTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const isMountedRef = useRef(true)

  useEffect(() => {
    isMountedRef.current = true
    return () => { isMountedRef.current = false }
  }, [])

  useEffect(() => {
    fetch(snapshotUrl, { credentials: 'same-origin' })
      .then((res) => {
        if (res.status === 404) return null
        if (!res.ok) throw new Error('Failed to load drawing')
        return res.json()
      })
      .then((wrapper) => {
        if (wrapper?.data) {
          const parsed = typeof wrapper.data === 'string' ? JSON.parse(wrapper.data) : wrapper.data
          const restored = restore(parsed, null, null)
          setInitialData({
            elements: restored.elements || [],
            appState: restored.appState || {},
          })
        }
        setLoaded(true)
      })
      .catch((err) => {
        console.error('Error loading excalidraw drawing:', err)
        setLoaded(true)
      })
  }, [snapshotUrl])

  const handleChange = useCallback(
    (elements: readonly any[], appState: any, files: any) => {
      if (readOnly || !isMountedRef.current) return

      clearTimeout(saveTimerRef.current)
      saveTimerRef.current = setTimeout(async () => {
        if (!isMountedRef.current) return
        _saveStatus = 'saving'
        notifySaveStatus()
        try {
          const json = serializeAsJSON(
            elements as any[],
            appState,
            files,
            'local',
          )
          const res = await fetch(saveUrl, {
            method: 'PUT',
            credentials: 'same-origin',
            headers: { 'Content-Type': 'application/json' },
            body: json,
          })
          if (!res.ok) throw new Error('Save failed')
          if (!isMountedRef.current) return
          _saveStatus = 'saved'
          notifySaveStatus()

          const api = apiRef.current
          if (api) {
            const sceneElements = api.getSceneElements()
            const sceneAppState = api.getAppState()
            const sceneFiles = api.getFiles()
            if (sceneElements.length > 0) {
              exportToSvg({
                elements: sceneElements,
                appState: {
                  ...sceneAppState,
                  exportBackground: true,
                  viewBackgroundColor: '#fefefe',
                },
                files: sceneFiles,
                exportPadding: 16,
                skipInliningFonts: true,
              }).then((svgEl: SVGSVGElement) => {
                fetch(saveUrl + '/svg', {
                  method: 'PUT',
                  credentials: 'same-origin',
                  headers: { 'Content-Type': 'image/svg+xml' },
                  body: svgEl.outerHTML,
                }).catch((e) => console.warn('excalidraw svg upload:', e))
              }).catch((e) => console.warn('excalidraw exportToSvg:', e))
            }
          }
          clearTimeout(fadeTimerRef.current)
          fadeTimerRef.current = setTimeout(() => {
            if (isMountedRef.current) {
              _saveStatus = 'idle'
              notifySaveStatus()
            }
          }, 2500)
        } catch {
          if (isMountedRef.current) {
            _saveStatus = 'error'
            notifySaveStatus()
          }
        }
      }, 2000)
    },
    [saveUrl, readOnly],
  )

  const toggleFullscreen = useCallback(() => {
    if (!document.fullscreenElement) {
      container.requestFullscreen().catch(() => {})
    } else {
      document.exitFullscreen().catch(() => {})
    }
  }, [container])

  useEffect(() => {
    const onFsChange = () => {
      setIsFullscreen(document.fullscreenElement === container)
    }
    document.addEventListener('fullscreenchange', onFsChange)
    return () => document.removeEventListener('fullscreenchange', onFsChange)
  }, [container])

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
  }, [toggleFullscreen])

  useEffect(() => {
    return () => {
      clearTimeout(saveTimerRef.current)
      clearTimeout(fadeTimerRef.current)
    }
  }, [])

  if (!loaded) {
    return <div className="excalidraw-loading">Loading drawing...</div>
  }

  return (
    <div className="excalidraw-canvas-container" style={{ height: '100%' }}>
      {!readOnly && <SaveStatusIndicator />}
      <Excalidraw
        excalidrawAPI={(api) => {
          apiRef.current = api
          if (!readOnly) {
            requestAnimationFrame(() => {
              api.setActiveTool({ type: 'freedraw' })
            })
          }
        }}
        initialData={initialData || undefined}
        viewModeEnabled={readOnly}
        onChange={readOnly ? undefined : handleChange}
        UIOptions={{
          canvasActions: {
            export: false,
            saveAsImage: false,
            loadScene: false,
          },
        }}
      />
    </div>
  )
}

declare global {
  interface Window {
    initExcalidrawIsland: (
      container: HTMLElement,
      snapshotUrl: string,
      saveUrl: string,
      options?: { readOnly?: boolean }
    ) => () => void
  }
}

window.initExcalidrawIsland = function (
  container: HTMLElement,
  snapshotUrl: string,
  saveUrl: string,
  options?: { readOnly?: boolean }
): () => void {
  _saveStatus = 'idle'
  _saveStatusListeners = []

  const root = createRoot(container)
  root.render(
    <StrictMode>
      <ExcalidrawIsland
        snapshotUrl={snapshotUrl}
        saveUrl={saveUrl}
        readOnly={options?.readOnly}
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
