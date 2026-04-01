import { createContext, useContext } from "react";
import type { Session } from "../types";
import { useSSE } from "../hooks/useSSE";

interface SessionsContextValue {
  sessions: Session[];
  connected: boolean;
  updateKey: number;
}

const SessionsContext = createContext<SessionsContextValue>({
  sessions: [],
  connected: false,
  updateKey: 0,
});

export function SessionsProvider({ children }: { children: React.ReactNode }) {
  const value = useSSE();
  return (
    <SessionsContext.Provider value={value}>
      {children}
    </SessionsContext.Provider>
  );
}

export function useSessions() {
  return useContext(SessionsContext);
}
