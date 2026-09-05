import { createContext, useContext, useEffect, useState, ReactNode } from "react";
import { api } from "./api";

export type User = { id: string; email: string; display_name: string; role: string };
export type Team = { id: string; name: string; invite_code?: string; role?: string };

type AuthCtx = {
  user: User | null;
  team: Team | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, display_name: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshTeam: () => Promise<void>;
  isAdmin: boolean;
};

const Ctx = createContext<AuthCtx>(null!);
export const useAuth = () => useContext(Ctx);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [team, setTeam] = useState<Team | null>(null);
  const [loading, setLoading] = useState(true);

  const loadTeam = async () => {
    try {
      const r = await api<{ team: Team | null }>("/me/team");
      setTeam(r.team);
    } catch {
      setTeam(null);
    }
  };

  useEffect(() => {
    api<User>("/auth/me")
      .then(async (u) => { setUser(u); await loadTeam(); })
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  const afterAuth = async (u: User) => { setUser(u); await loadTeam(); };
  const login = async (email: string, password: string) => {
    await afterAuth(await api<User>("/auth/login", { method: "POST", body: JSON.stringify({ email, password }) }));
  };
  const register = async (email: string, display_name: string, password: string) => {
    await afterAuth(await api<User>("/auth/register", { method: "POST", body: JSON.stringify({ email, display_name, password }) }));
  };
  const logout = async () => {
    await api("/auth/logout", { method: "POST" });
    setUser(null); setTeam(null);
  };

  return (
    <Ctx.Provider value={{
      user, team, loading, login, register, logout, refreshTeam: loadTeam,
      isAdmin: !!user && (user.role === "owner" || user.role === "admin"),
    }}>
      {children}
    </Ctx.Provider>
  );
}
