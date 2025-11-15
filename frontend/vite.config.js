import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// Read version from version.json
let version = 'unknown'
try {
  const versionFile = readFileSync(resolve(__dirname, 'version.json'), 'utf-8')
  const versionData = JSON.parse(versionFile)
  version = versionData.version
} catch (error) {
  console.warn('Warning: Could not read version.json, using default version')
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  define: {
    __APP_VERSION__: JSON.stringify(version),
  },
})
