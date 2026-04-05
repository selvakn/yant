import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    lib: {
      entry: resolve(__dirname, 'src/tldraw-island.tsx'),
      name: 'TldrawIsland',
      fileName: 'tldraw-bundle',
      formats: ['iife'],
    },
    outDir: resolve(__dirname, '../frontend/static/vendor'),
    emptyOutDir: false,
    rollupOptions: {
      output: {
        assetFileNames: 'tldraw-bundle.[ext]',
        entryFileNames: 'tldraw-bundle.js',
      },
    },
  },
})
