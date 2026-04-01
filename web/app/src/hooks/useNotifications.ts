import { useEffect, useRef } from "react";
import type { Session } from "../types";
import { countAttention, formatTitle } from "../domain/session";
import {
  detectTransitions,
  buildStateMap,
  type StateTransition,
} from "../domain/notifications";

export function useNotifications(sessions: Session[]) {
  const prevStates = useRef<Map<string, string>>(new Map());
  const initialized = useRef(false);
  const permissionRequested = useRef(false);

  const attentionCount = countAttention(sessions);

  useEffect(() => {
    document.title = formatTitle(attentionCount);
  }, [attentionCount]);

  useEffect(() => {
    if (!initialized.current) {
      prevStates.current = buildStateMap(sessions);
      initialized.current = true;
      return;
    }

    const transitions = detectTransitions(prevStates.current, sessions);
    for (const t of transitions) {
      fireNotification(t);
    }

    prevStates.current = buildStateMap(sessions);
  }, [sessions]);

  useEffect(() => {
    if (permissionRequested.current) return;

    function handler() {
      permissionRequested.current = true;
      if ("Notification" in window && Notification.permission === "default") {
        Notification.requestPermission();
      }
    }

    document.addEventListener("click", handler, { once: true });
    return () => document.removeEventListener("click", handler);
  }, [sessions]);

  return { attentionCount };
}

const titles: Record<string, string> = {
  needs_input: "needs attention",
  finished: "finished",
  ended: "session ended",
};

function fireNotification(t: StateTransition) {
  if (!("Notification" in window) || Notification.permission !== "granted")
    return;
  new Notification(`${t.name} — ${titles[t.kind] ?? t.kind}`, {
    body: t.executor_detail,
    icon: "/icon.svg",
    tag: `cctl-${t.name}-${t.kind}`,
  });
}
