import {
  type ReactNode,
  createContext,
  useContext,
  useEffect,
  useState,
} from 'react'

export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'theme'

/**
 * Injected verbatim into <head> (see routes/__root.tsx) so the right theme class
 * is on <html> before first paint. Prevents a light-mode flash and keeps the
 * server markup in sync with what the client hydrates.
 */
export const themeInitScript = `(function () {
  try {
    var stored = localStorage.getItem('${STORAGE_KEY}')
    var dark = stored
      ? stored === 'dark'
      : window.matchMedia('(prefers-color-scheme: dark)').matches
    document.documentElement.classList.toggle('dark', dark)
  } catch (e) {}
})()`

function readTheme(): Theme {
  if (typeof document === 'undefined') return 'light'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => void
  toggleTheme: () => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

export function ThemeProvider({ children }: { children: ReactNode }) {
  // The init script has already applied the class by the time this runs on the
  // client; on the server we fall back to 'light'.
  const [theme, setTheme] = useState<Theme>(readTheme)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    try {
      localStorage.setItem(STORAGE_KEY, theme)
    } catch {
      // Ignore storage failures (private mode, disabled cookies, ...).
    }
  }, [theme])

  const value: ThemeContextValue = {
    theme,
    setTheme,
    toggleTheme: () => setTheme(theme === 'dark' ? 'light' : 'dark'),
  }

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return ctx
}
