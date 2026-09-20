import type { Dispatch,ReactNode,SetStateAction } from "react";
import {
createContext,
useCallback,
useContext,
useEffect,
useState,
} from "react";
import { api } from "../api/client";
import type { Section,Session } from "../api/types";
import { useLoad } from "../hooks/useLoad";

type AppContextValue = {
  session: Session;
  setSession: Dispatch<SetStateAction<Session>>;
  ensureSession: () => Promise<Session>;
  sections: Section[];
  sectionsError: string;
  reloadSections: () => void;
  refresh: () => void;
  version: number;
  notify: (text: string) => void;
  toast: string;
  dismissToast: () => void;
};

const AppContext = createContext<AppContextValue | null>(null);

export function useApp() {
  const value = useContext(AppContext);
  if (!value) throw new Error("useApp must be used inside AppProvider");
  return value;
}

export function AppProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session>({
    active: false,
    csrf_token: "",
    staff: null,
  });
  const [version, setVersion] = useState(0);
  const [toast, setToast] = useState("");
  const sectionsLoad = useLoad<{ items: Section[] }>("/sections", version);

  useEffect(() => {
    const controller = new AbortController();
    api<Session>("/session", { signal: controller.signal })
      .then((session) => {
        if (!controller.signal.aborted) setSession(session);
      })
      .catch(() => {});
    return () => controller.abort();
  }, []);

  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(() => setToast(""), 5000);
    return () => clearTimeout(timer);
  }, [toast]);

  const ensureSession = useCallback(async () => {
    const session = await api<Session>("/sessions", { method: "POST" });
    setSession(session);
    return session;
  }, []);
  const refresh = useCallback(() => setVersion((version) => version + 1), []);
  const dismissToast = useCallback(() => setToast(""), []);

  return (
    <AppContext.Provider
      value={{
        session,
        setSession,
        ensureSession,
        sections: sectionsLoad.data?.items ?? [],
        sectionsError: sectionsLoad.error,
        reloadSections: sectionsLoad.reload,
        version,
        refresh,
        notify: setToast,
        toast,
        dismissToast,
      }}
    >
      {children}
    </AppContext.Provider>
  );
}
