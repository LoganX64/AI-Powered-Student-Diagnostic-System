import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  fetchSession,
  invalidateSession,
  type Session,
} from "@/lib/session";
import { ROLE_CHANGE_EVENT } from "@/hooks/useRole";

export type SessionStatus =
  | "loading"
  | "authenticated"
  | "anonymous"
  | "error";

type SessionContextValue = {
  status: SessionStatus;
  session: Session | null;
  refresh: () => Promise<void>;
};

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>("loading");
  const [session, setSession] = useState<Session | null>(null);
  const requestId = useRef(0);

  const applySession = useCallback(async (force: boolean) => {
    const id = ++requestId.current;
    setStatus((current) => (current === "error" ? "loading" : current));

    try {
      const next = await fetchSession(force);
      if (id !== requestId.current) return;
      setSession(next);
      setStatus(next ? "authenticated" : "anonymous");
    } catch {
      if (id !== requestId.current) return;
      setSession(null);
      setStatus("error");
    }
  }, []);

  const refresh = useCallback(async () => {
    await applySession(true);
  }, [applySession]);

  useEffect(() => {
    void applySession(false);

    const handleRoleChange = () => {
      invalidateSession();
      void applySession(true);
    };
    window.addEventListener(ROLE_CHANGE_EVENT, handleRoleChange);

    return () => {
      requestId.current += 1;
      window.removeEventListener(ROLE_CHANGE_EVENT, handleRoleChange);
    };
  }, [applySession]);

  const value = useMemo(
    () => ({ status, session, refresh }),
    [status, session, refresh],
  );

  return (
    <SessionContext.Provider value={value}>
      {children}
    </SessionContext.Provider>
  );
}

export function useSession(): SessionContextValue {
  const value = useContext(SessionContext);
  if (!value) {
    throw new Error("useSession must be used within a SessionProvider");
  }
  return value;
}
