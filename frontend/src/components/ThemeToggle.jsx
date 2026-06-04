import { useTheme } from '../contexts/ThemeContext'

const THEME_LABELS = {
  light: 'Light mode',
  dark: 'Dark mode',
  system: 'System theme',
}

const NEXT_THEME_LABELS = {
  light: 'dark mode',
  dark: 'system theme',
  system: 'light mode',
}

function ThemeToggle({ variant = 'header' }) {
  const { theme, resolvedTheme, toggle } = useTheme()
  const variantClass =
    variant === 'surface' ? 'theme-toggle-btn-surface' : 'theme-toggle-btn'

  const title = `${THEME_LABELS[theme]}${theme === 'system' ? ` (${resolvedTheme})` : ''}. Switch to ${NEXT_THEME_LABELS[theme]}.`

  return (
    <button
      type="button"
      onClick={toggle}
      className={`btn btn-sm ${variantClass} p-1 d-inline-flex align-items-center justify-content-center`}
      title={title}
      aria-label={title}
    >
      {theme === 'dark' ? (
        <svg
          width="1.1rem"
          height="1.1rem"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.25"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z" />
        </svg>
      ) : theme === 'light' ? (
        <svg
          width="1.1rem"
          height="1.1rem"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.25"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="5" />
          <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
        </svg>
      ) : (
        <svg
          width="1.1rem"
          height="1.1rem"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.25"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
          <path d="M8 21h8M12 17v4" />
        </svg>
      )}
    </button>
  )
}

export default ThemeToggle
