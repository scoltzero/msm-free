import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api, clearSession, getToken, setSession } from "@/lib/api";

export interface CurrentUser {
  id?: number | string;
  username: string;
  role?: string;
  email?: string;
  display_name?: string;
  is_active?: boolean;
}

interface AuthContextValue {
  loading: boolean;
  initialized: boolean;
  setupNeedsRecovery: boolean;
  setupDownloadComponents: string[];
  user: CurrentUser | null;
  refresh: () => Promise<void>;
  login: (username: string, password: string) => Promise<CurrentUser>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [loading, setLoading] = useState(true);
  const [initialized, setInitialized] = useState(false);
  const [setupNeedsRecovery, setSetupNeedsRecovery] = useState(false);
  const [setupDownloadComponents, setSetupDownloadComponents] = useState<string[]>([]);
  const [user, setUser] = useState<CurrentUser | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const setup = await api<any>("/api/v1/setup/check", { skipAuth: true });
      const ready = !!setup?.is_initialized;
      const components = Array.isArray(setup?.download_component)
        ? setup.download_component.map((item: unknown) => String(item)).filter(Boolean)
        : [];
      setInitialized(ready);
      setSetupNeedsRecovery(Boolean(setup?.needs_recovery || setup?.needs_download));
      setSetupDownloadComponents(components);
      if (ready && getToken()) {
        const me = await api<any>("/api/v1/auth/me");
        setUser(me.user || me.data || null);
      } else {
        setUser(null);
      }
    } catch {
      setInitialized(false);
      setSetupNeedsRecovery(false);
      setSetupDownloadComponents([]);
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const onExpired = () => setUser(null);
    window.addEventListener("msf-auth-expired", onExpired);
    return () => window.removeEventListener("msf-auth-expired", onExpired);
  }, [refresh]);

  const login = useCallback(async (username: string, password: string) => {
    const payload = await api<any>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
      skipAuth: true,
    });
    setSession(payload.token, payload.refresh_token);
    const nextUser = payload.user || payload.data?.user || { username };
    setUser(nextUser);
    setInitialized(true);
    setSetupNeedsRecovery(false);
    setSetupDownloadComponents([]);
    return nextUser;
  }, []);

  const logout = useCallback(() => {
    clearSession();
    setUser(null);
  }, []);

  const value = useMemo(
    () => ({ loading, initialized, setupNeedsRecovery, setupDownloadComponents, user, refresh, login, logout }),
    [loading, initialized, setupNeedsRecovery, setupDownloadComponents, user, refresh, login, logout]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return value;
}
