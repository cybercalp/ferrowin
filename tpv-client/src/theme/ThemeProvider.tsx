import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  type ReactNode,
} from "react";

/* ------------------------------------------------------------------------
   Types
   ------------------------------------------------------------------------ */
type Theme = "light" | "dark";

interface ThemeContextValue {
  theme: Theme;
  toggleTheme: () => void;
}

/* ------------------------------------------------------------------------
   Context
   ------------------------------------------------------------------------ */
const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

/* ------------------------------------------------------------------------
   Provider
   ------------------------------------------------------------------------ */
function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    // Read the attribute set by the blocking script in index.html
    const attr = document.documentElement.getAttribute("data-theme");
    return attr === "dark" ? "dark" : "light";
  });

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem("tpv-theme", theme);
  }, [theme]);

  const toggleTheme = useCallback(() => {
    setTheme((prev) => (prev === "light" ? "dark" : "light"));
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

/* ------------------------------------------------------------------------
   Hook
   ------------------------------------------------------------------------ */
function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within a <ThemeProvider>");
  }
  return ctx;
}

/* ------------------------------------------------------------------------
   Toggle Button Component
   ------------------------------------------------------------------------ */
function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();

  return (
    <button
      onClick={toggleTheme}
      style={{
        background: "var(--bg-card)",
        border: "1px solid var(--border-default)",
        borderRadius: "8px",
        padding: "8px 14px",
        cursor: "pointer",
        color: "var(--text-secondary)",
        fontSize: "13px",
        fontWeight: "600",
        transition: "all 0.2s",
        display: "inline-flex",
        alignItems: "center",
        gap: "6px",
      }}
      aria-label={
        theme === "light" ? "Switch to dark mode" : "Switch to light mode"
      }
    >
      <span>{theme === "light" ? "🌙" : "☀️"}</span>
      <span>{theme === "light" ? "Dark" : "Light"}</span>
    </button>
  );
}

export { ThemeProvider, ThemeToggle, useTheme };
export type { Theme, ThemeContextValue };
