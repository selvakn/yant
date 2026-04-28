import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

const outDir = resolve(__dirname, '../frontend/static/vendor')

// Vite multi-entry IIFE build: two separate island bundles.
// Selected via BUNDLE env var when building individually,
// or built sequentially by the npm build script.
const bundle = process.env.BUNDLE || 'tldraw'

const entries: Record<string, { entry: string; name: string; fileName: string }> = {
  tldraw: {
    entry: resolve(__dirname, 'src/tldraw-island.tsx'),
    name: 'TldrawIsland',
    fileName: 'tldraw-bundle',
  },
  excalidraw: {
    entry: resolve(__dirname, 'src/excalidraw-island.tsx'),
    name: 'ExcalidrawIsland',
    fileName: 'excalidraw-bundle',
  },
}

const active = entries[bundle]

export default defineConfig({
  plugins: [react()],
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    lib: {
      entry: active.entry,
      name: active.name,
      fileName: active.fileName,
      formats: ['iife'],
    },
    outDir,
    emptyOutDir: false,
    rollupOptions: {
      output: {
        assetFileNames: `${active.fileName}.[ext]`,
        entryFileNames: `${active.fileName}.js`,
      },
    },
  },
})
