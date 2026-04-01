import { useEffect, useRef, useMemo, useState, useCallback } from "react";
import type { Session } from "../../types";
import { useTranscript } from "../../hooks/useTranscript";
import { pairedResultIds, findToolResult } from "../../domain/transcript";
import { isActiveDetail } from "../../domain/session";
import { AssistantBubble } from "./AssistantBubble";
import { ToolCall } from "./ToolCall";
import { TypingIndicator } from "./TypingIndicator";
import { ActionBar } from "./ActionBar";
import { ReplyBar } from "./ReplyBar";

interface SessionViewProps {
  session: Session | undefined;
  sessionId: string;
  updateKey: number;
}

export function SessionView({ session, sessionId, updateKey }: SessionViewProps) {
  const { entries, loading } = useTranscript(sessionId, updateKey);
  const chatRef = useRef<HTMLDivElement>(null);
  const wasAtBottom = useRef(true);
  const [pendingMessages, setPendingMessages] = useState<string[]>([]);

  const handleSent = useCallback((text: string) => {
    setPendingMessages(prev => [...prev, text]);
  }, []);

  const needsInitialScroll = useRef(true);

  // Clear pending messages and reset scroll on session switch
  useEffect(() => {
    setPendingMessages([]);
    needsInitialScroll.current = true;
    wasAtBottom.current = true;
  }, [sessionId]);

  // Remove pending messages that now appear in the transcript
  useEffect(() => {
    if (pendingMessages.length === 0) return;
    const transcriptTexts = new Set(
      entries.filter(e => e.role === "user").map(e => e.text),
    );
    const remaining = pendingMessages.filter(m => !transcriptTexts.has(m));
    if (remaining.length < pendingMessages.length) {
      setPendingMessages(remaining);
    }
  }, [entries, pendingMessages]);

  // Track scroll position via event handler (avoids render-order race)
  const handleScroll = useCallback(() => {
    const el = chatRef.current;
    if (!el) return;
    wasAtBottom.current =
      el.scrollHeight - el.scrollTop - el.clientHeight < 60;
  }, []);

  // Scroll to bottom on initial load or when new content arrives
  useEffect(() => {
    const el = chatRef.current;
    if (!el) return;
    if (needsInitialScroll.current || wasAtBottom.current) {
      el.scrollTop = el.scrollHeight;
      needsInitialScroll.current = false;
    }
  }, [entries, pendingMessages]);

  const paired = useMemo(() => pairedResultIds(entries), [entries]);

  return (
    <div className="h-full flex flex-col">
      <div ref={chatRef} onScroll={handleScroll} className="flex-1 overflow-y-auto p-4 space-y-3">
        {loading && entries.length === 0 ? (
          <div className="text-center text-gray-500 py-8 text-sm">
            Loading...
          </div>
        ) : entries.length === 0 ? (
          <div className="text-center text-gray-500 py-8 text-sm">
            No transcript available
          </div>
        ) : (
          entries.map((entry, i) => {
            switch (entry.role) {
              case "user":
                return (
                  <div key={i} className="flex justify-end">
                    <div className="max-w-[85%] bg-blue-600/10 dark:bg-blue-600/15 text-gray-900 dark:text-gray-100 text-sm px-3 py-2 rounded-xl rounded-br-sm">
                      {entry.text}
                    </div>
                  </div>
                );
              case "assistant":
                return <AssistantBubble key={i} text={entry.text} />;
              case "tool_use": {
                const toolResult = findToolResult(entries, i);
                const hasSubsequent = i < entries.length - 2;
                return (
                  <ToolCall
                    key={i}
                    entry={entry}
                    result={toolResult}
                    cwd={session?.cwd}
                    pending={!toolResult && !hasSubsequent}
                  />
                );
              }
              case "tool_result":
                if (entry.tool_use_id && paired.has(entry.tool_use_id)) {
                  return null;
                }
                return (
                  <div
                    key={i}
                    className={`border-l-2 ${entry.is_error ? "border-red-500/40" : "border-gray-300 dark:border-gray-700"} pl-3 font-mono text-xs text-gray-500 whitespace-pre-wrap break-all max-h-24 overflow-hidden`}
                  >
                    {entry.text}
                  </div>
                );
              default:
                return null;
            }
          })
        )}

        {pendingMessages.map((msg, i) => (
          <div key={`pending-${i}`} className="flex justify-end">
            <div className="max-w-[85%] bg-blue-600/10 dark:bg-blue-600/15 text-gray-900 dark:text-gray-100 text-sm px-3 py-2 rounded-xl rounded-br-sm">
              {msg}
            </div>
          </div>
        ))}

        {session?.executor_state === "WORKING" && (
          <TypingIndicator detail={isActiveDetail(session.executor_detail) ? session.executor_detail : undefined} />
        )}

        {pendingMessages.length > 0 && session?.executor_state !== "WORKING" && (
          <div className="flex justify-start">
            <div className="text-xs text-gray-400 dark:text-gray-500 px-3 py-1">
              Sent — waiting for response...
            </div>
          </div>
        )}
      </div>

      {session && <ActionBar session={session} />}
      <ReplyBar sessionId={sessionId} sessionState={session?.executor_state} attached={session?.attached} onSent={handleSent} />
    </div>
  );
}
