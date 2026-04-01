import { useEffect, useRef, useState, useCallback } from "react";
import type { Session } from "../types";

export function useSSE() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [connected, setConnected] = useState(false);
  const [updateKey, setUpdateKey] = useState(0);
  const retryDelay = useRef(1000);
  const esRef = useRef<EventSource | null>(null);

  const connect = useCallback(() => {
    const es = new EventSource("/api/events");
    esRef.current = es;

    es.addEventListener("sessions", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as Session[] | null;
        setSessions(data ?? []);
        setUpdateKey((k) => k + 1);
        retryDelay.current = 1000;
      } catch {
        // ignore parse errors
      }
    });

    es.onopen = () => {
      setConnected(true);
      retryDelay.current = 1000;
    };

    es.onerror = () => {
      setConnected(false);
      es.close();
      esRef.current = null;
      const delay = retryDelay.current;
      retryDelay.current = Math.min(delay * 2, 30000);
      setTimeout(connect, delay);
    };
  }, []);

  useEffect(() => {
    connect();
    return () => {
      esRef.current?.close();
      esRef.current = null;
    };
  }, [connect]);

  return { sessions, connected, updateKey };
}
