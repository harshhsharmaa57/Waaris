import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import {
  auth,
  setTokens,
  clearTokens,
  getRefreshToken,
  type SessionResponse,
} from "../lib/api";

interface User {
  id: string;
  email: string;
  displayName: string;
  createdAt: string;
}

interface AuthState {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, displayName?: string) => Promise<void>;
  logout: () => Promise<void>;
  updateProfile: (displayName: string) => Promise<void>;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const rt = getRefreshToken();
    if (!rt) {
      setLoading(false);
      return;
    }
    auth
      .refresh(rt)
      .then((s: SessionResponse) => {
        setTokens(s.accessToken, s.refreshToken);
        setUser(s.user);
      })
      .catch(() => clearTokens())
      .finally(() => setLoading(false));
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const s = await auth.login(email, password);
    setTokens(s.accessToken, s.refreshToken);
    setUser(s.user);
  }, []);

  const register = useCallback(
    async (email: string, password: string, displayName?: string) => {
      const s = await auth.register(email, password, displayName);
      setTokens(s.accessToken, s.refreshToken);
      setUser(s.user);
    },
    [],
  );

  const logout = useCallback(async () => {
    const rt = getRefreshToken();
    if (rt) {
      try {
        await auth.logout(rt);
      } catch {
        /* best effort */
      }
    }
    clearTokens();
    setUser(null);
  }, []);

  const updateProfile = useCallback(async (displayName: string) => {
    const u = await auth.updateMe(displayName);
    setUser(u);
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout, updateProfile }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be inside AuthProvider");
  return ctx;
}
