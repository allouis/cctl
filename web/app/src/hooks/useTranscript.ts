import { useState, useEffect, useRef } from "react";
import type { TranscriptEntry } from "../types";
import { getTranscript } from "../api/client";

export function useTranscript(sessionId: string | null, updateKey: number) {
  const [entries, setEntries] = useState<TranscriptEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const prevId = useRef<string | null>(null);

  useEffect(() => {
    if (!sessionId) {
      setEntries([]);
      return;
    }

    const isNewSession = prevId.current !== sessionId;
    prevId.current = sessionId;

    if (isNewSession) {
      setLoading(true);
    }

    let cancelled = false;
    getTranscript(sessionId)
      .then((data) => {
        if (!cancelled) {
          setEntries(data);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setEntries([]);
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [sessionId, updateKey]);

  return { entries, loading };
}
