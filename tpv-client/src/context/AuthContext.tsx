import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import { invoke } from "@tauri-apps/api/core";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface UserInfo {
  id: string;
  username: string;
}

interface AuthContextValue {
  user: UserInfo | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
}

const LOCAL_STORAGE_TOKEN_KEY = "ferrowin_token";
const LOCAL_STORAGE_USER_KEY = "ferrowin_user";

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const isAuthenticated = token !== null;

  // Simple JWT payload decode (no signature verification — client-side only for expiry check)
  function decodeTokenPayload(t: string): Record<string, unknown> | null {
    try {
      const parts = t.split(".");
      if (parts.length !== 3) return null;
      const payload = parts[1];
      const decoded = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
      return JSON.parse(decoded);
    } catch {
      return null;
    }
  }

  const login = useCallback(async (username: string, password: string) => {
    setError(null);
    try {
      const userInfo = await invoke<UserInfo>("login", { username, password });
      // After successful login, the Rust AuthState is already updated.
      // We need to read the token back from Rust or reconstruct it.
      // The Rust `login` command returns UserInfo but not the token directly
      // to the frontend in the current design. Let's store the token separately.

      // Actually, looking at the Rust login command: it returns UserInfo.
      // But the frontend also needs the token for localStorage persistence.
      // The Rust command stores the token internally. We need the token in frontend too.
      //
      // Let's fetch the stored token from Rust. Actually, there's no command to
      // get the token back. Let me check the design again...
      //
      // The design says: "localStorage.set('ferrowin:auth_token', token)" and
      // "invoke('set_auth_token', {token})".
      //
      // This means the login command should also return the token to the frontend.
      // But our Rust login command only returns UserInfo.
      //
      // Solution: We'll modify the approach — the frontend will pass the token
      // back to Rust after login. But we need the token first.
      //
      // Let me invoke a new command to retrieve the current token from Rust state.

      const currentToken = await invoke<string>("get_auth_token");
      setToken(currentToken);
      setUser(userInfo);
      localStorage.setItem(LOCAL_STORAGE_TOKEN_KEY, currentToken);
      localStorage.setItem(LOCAL_STORAGE_USER_KEY, JSON.stringify(userInfo));
    } catch (err) {
      const message =
        typeof err === "string"
          ? err
          : "Invalid credentials";
      setError(message);
      throw err;
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await invoke("clear_auth");
    } catch {
      // ignore errors during logout
    }
    setToken(null);
    setUser(null);
    setError(null);
    localStorage.removeItem(LOCAL_STORAGE_TOKEN_KEY);
    localStorage.removeItem(LOCAL_STORAGE_USER_KEY);
  }, []);

  const checkAuth = useCallback(async () => {
    setIsLoading(true);
    try {
      const storedToken = localStorage.getItem(LOCAL_STORAGE_TOKEN_KEY);
      const storedUserRaw = localStorage.getItem(LOCAL_STORAGE_USER_KEY);

      if (!storedToken || !storedUserRaw) {
        setIsLoading(false);
        return;
      }

      // Decode JWT client-side to check expiry
      const payload = decodeTokenPayload(storedToken);
      if (!payload) {
        // Invalid token, clear and show login
        localStorage.removeItem(LOCAL_STORAGE_TOKEN_KEY);
        localStorage.removeItem(LOCAL_STORAGE_USER_KEY);
        setIsLoading(false);
        return;
      }

      // Check expiration
      const exp = payload.exp as number | undefined;
      if (exp && Date.now() >= exp * 1000) {
        // Token expired
        localStorage.removeItem(LOCAL_STORAGE_TOKEN_KEY);
        localStorage.removeItem(LOCAL_STORAGE_USER_KEY);
        setIsLoading(false);
        return;
      }

      // Token is still valid — restore auth state in Rust and frontend
      let storedUser: UserInfo;
      try {
        storedUser = JSON.parse(storedUserRaw);
      } catch {
        localStorage.removeItem(LOCAL_STORAGE_TOKEN_KEY);
        localStorage.removeItem(LOCAL_STORAGE_USER_KEY);
        setIsLoading(false);
        return;
      }

      await invoke("set_auth_state", {
        token: storedToken,
        userId: storedUser.id,
        username: storedUser.username,
      });

      setToken(storedToken);
      setUser(storedUser);
    } catch {
      // If anything fails (e.g., Tauri not available in dev mode), silently degrade
      console.warn("Auth check failed, falling back to unauthenticated state");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isAuthenticated,
        isLoading,
        error,
        login,
        logout,
        checkAuth,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
