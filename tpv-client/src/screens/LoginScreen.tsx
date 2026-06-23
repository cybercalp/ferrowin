import { useState, type FormEvent } from "react";
import { useAuth } from "../context/AuthContext";

export function LoginScreen() {
  const { login, error } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!username.trim() || !password) {
      setLocalError("Usuario y contraseña son requeridos");
      return;
    }

    setIsSubmitting(true);
    setLocalError(null);

    try {
      await login(username.trim(), password);
    } catch {
      // Error is handled by the context
      setLocalError(null); // error from context will display
    } finally {
      setIsSubmitting(false);
    }
  }

  const displayError = localError || error;

  return (
    <div style={styles.wrapper}>
      <div style={styles.container}>
        {/* Logo / Brand */}
        <div style={styles.logoSection}>
          <div style={styles.logoIcon}>
            <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
              <rect width="48" height="48" rx="12" fill="var(--primary, #2563eb)" />
              <path
                d="M14 28V18h4v10h-4zM22 28V14h4v14h-4zM30 28v-6h4v6h-4z"
                fill="#fff"
              />
            </svg>
          </div>
          <h1 style={styles.title}>Ferrowin TPV</h1>
          <p style={styles.subtitle}>Inicia sesión para continuar</p>
        </div>

        {/* Login Form */}
        <form onSubmit={handleSubmit} style={styles.form}>
          <div style={styles.inputGroup}>
            <label htmlFor="username" style={styles.label}>
              Usuario
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Ingrese su usuario"
              style={styles.input}
              disabled={isSubmitting}
              autoFocus
              autoComplete="username"
            />
          </div>

          <div style={styles.inputGroup}>
            <label htmlFor="password" style={styles.label}>
              Contraseña
            </label>
            <div style={styles.passwordWrapper}>
              <input
                id="password"
                type={showPassword ? "text" : "password"}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Ingrese su contraseña"
                style={styles.input}
                disabled={isSubmitting}
                autoComplete="current-password"
              />
              <button
                type="button"
                style={styles.togglePassword}
                onClick={() => setShowPassword(!showPassword)}
                tabIndex={-1}
                aria-label={showPassword ? "Ocultar contraseña" : "Mostrar contraseña"}
              >
                {showPassword ? (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                    <line x1="1" y1="1" x2="23" y2="23" />
                  </svg>
                ) : (
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                )}
              </button>
            </div>
          </div>

          {displayError && (
            <div style={styles.errorBox}>
              <span style={styles.errorText}>{displayError}</span>
            </div>
          )}

          <button
            type="submit"
            style={{
              ...styles.submitButton,
              ...(isSubmitting ? styles.submitButtonDisabled : {}),
            }}
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <span style={styles.loadingSpinner}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" style={{ animation: "spin 1s linear infinite" }}>
                  <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" opacity="0.25" />
                  <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" strokeWidth="4" strokeLinecap="round" />
                </svg>
                Iniciando sesión...
              </span>
            ) : (
              "Iniciar Sesión"
            )}
          </button>
        </form>

        <style>{`
          @keyframes spin {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
          }
        `}</style>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const styles: Record<string, React.CSSProperties> = {
  wrapper: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    minHeight: "100vh",
    width: "100%",
    backgroundColor: "var(--bg-primary, #f8fafc)",
    padding: "16px",
    boxSizing: "border-box",
  },
  container: {
    width: "100%",
    maxWidth: "400px",
    backgroundColor: "var(--bg-secondary, #ffffff)",
    borderRadius: "16px",
    boxShadow: "var(--shadow-lg, 0 10px 25px rgba(0,0,0,0.1))",
    padding: "40px 32px",
    boxSizing: "border-box",
  },
  logoSection: {
    textAlign: "center",
    marginBottom: "32px",
  },
  logoIcon: {
    marginBottom: "16px",
    display: "flex",
    justifyContent: "center",
  },
  title: {
    fontSize: "24px",
    fontWeight: "700",
    color: "var(--text-primary, #1e293b)",
    margin: "0 0 8px 0",
    fontFamily: "var(--font-family, system-ui)",
  },
  subtitle: {
    fontSize: "14px",
    color: "var(--text-secondary, #64748b)",
    margin: 0,
    fontFamily: "var(--font-family, system-ui)",
  },
  form: {
    display: "flex",
    flexDirection: "column",
    gap: "20px",
  },
  inputGroup: {
    display: "flex",
    flexDirection: "column",
    gap: "6px",
  },
  label: {
    fontSize: "13px",
    fontWeight: "600",
    color: "var(--text-primary, #1e293b)",
    fontFamily: "var(--font-family, system-ui)",
  },
  input: {
    width: "100%",
    padding: "10px 12px",
    fontSize: "15px",
    border: "1px solid var(--border-color, #e2e8f0)",
    borderRadius: "8px",
    backgroundColor: "var(--bg-input, #ffffff)",
    color: "var(--text-primary, #1e293b)",
    outline: "none",
    boxSizing: "border-box",
    fontFamily: "var(--font-family, system-ui)",
    transition: "border-color 0.15s ease",
  },
  passwordWrapper: {
    position: "relative",
    display: "flex",
    alignItems: "center",
  },
  togglePassword: {
    position: "absolute",
    right: "8px",
    top: "50%",
    transform: "translateY(-50%)",
    background: "none",
    border: "none",
    cursor: "pointer",
    padding: "4px",
    color: "var(--text-secondary, #64748b)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
  errorBox: {
    backgroundColor: "var(--error-bg, #fef2f2)",
    border: "1px solid var(--error-border, #fecaca)",
    borderRadius: "8px",
    padding: "10px 14px",
  },
  errorText: {
    color: "var(--error-text, #dc2626)",
    fontSize: "13px",
    fontWeight: "500",
    fontFamily: "var(--font-family, system-ui)",
  },
  submitButton: {
    width: "100%",
    padding: "12px 16px",
    fontSize: "15px",
    fontWeight: "600",
    color: "#ffffff",
    backgroundColor: "var(--primary, #2563eb)",
    border: "none",
    borderRadius: "8px",
    cursor: "pointer",
    fontFamily: "var(--font-family, system-ui)",
    transition: "background-color 0.15s ease",
  },
  submitButtonDisabled: {
    opacity: 0.7,
    cursor: "not-allowed",
  },
  loadingSpinner: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    gap: "8px",
  },
};
