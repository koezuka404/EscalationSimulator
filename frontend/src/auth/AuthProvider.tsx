import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { login as loginRequest, logout as logoutRequest, refreshSession, setAccessToken } from "../api/client";
import type { User } from "../api/types";

type AuthValue = {
  user: User | null;
  ready: boolean;
  login: (email: string, password: string) => Promise<User>;
  logout: () => Promise<void>;
  replaceUser: (user: User) => void;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let ignore = false;
    refreshSession()
      .then((result) => {
        if (ignore) return;
        setUser(result?.user ?? null);
      })
      .finally(() => {
        if (!ignore) setReady(true);
      });
    return () => {
      ignore = true;
    };
  }, []);

  const value = useMemo<AuthValue>(
    () => ({
      user,
      ready,
      async login(email, password) {
        const result = await loginRequest(email, password);
        setUser(result.user);
        return result.user;
      },
      async logout() {
        await logoutRequest();
        setAccessToken("");
        setUser(null);
      },
      replaceUser(next) {
        setUser(next);
      },
    }),
    [user, ready],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("ログイン状態を画面の外から使っています");
  return value;
}
