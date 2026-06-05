import { createContext, useContext, useState, useEffect, useCallback } from 'react'

const STORAGE_KEY = 'sbv-theme'
const THEME_ORDER = ['light', 'dark', 'system']

const ThemeContext = createContext(null)

function getSystemTheme() {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light'
  }
  return 'light'
}

function resolveTheme(preference) {
  if (preference === 'system') {
    return getSystemTheme()
  }
  return preference
}

function readStoredPreference() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'dark' || saved === 'light' || saved === 'system') {
      return saved
    }
  } catch {
    // localStorage unavailable (e.g. private mode on some browsers)
  }
  return 'system'
}

export function ThemeProvider({ children }) {
  const [theme, setTheme] = useState(readStoredPreference)
  const [resolvedTheme, setResolvedTheme] = useState(() =>
    resolveTheme(readStoredPreference()),
  )

  const applyResolvedTheme = useCallback((resolved) => {
    setResolvedTheme(resolved)
    document.documentElement.setAttribute('data-bs-theme', resolved)
  }, [])

  useEffect(() => {
    applyResolvedTheme(resolveTheme(theme))
  }, [theme, applyResolvedTheme])

  useEffect(() => {
    if (theme !== 'system' || !window.matchMedia) {
      return undefined
    }

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => applyResolvedTheme(getSystemTheme())

    mediaQuery.addEventListener('change', onChange)
    return () => mediaQuery.removeEventListener('change', onChange)
  }, [theme, applyResolvedTheme])

  const toggle = () => {
    setTheme((current) => {
      const index = THEME_ORDER.indexOf(current)
      const next = THEME_ORDER[(index + 1) % THEME_ORDER.length]
      try {
        localStorage.setItem(STORAGE_KEY, next)
      } catch {
        // ignore storage errors
      }
      return next
    })
  }

  return (
    <ThemeContext.Provider value={{ theme, resolvedTheme, toggle }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}
