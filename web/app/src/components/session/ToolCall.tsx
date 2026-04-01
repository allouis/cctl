import { useState } from "react";
import type { TranscriptEntry } from "../../types";
import { Spinner } from "../shared/Spinner";
import { shortenToolText } from "../../domain/transcript";

interface ToolCallProps {
  entry: TranscriptEntry;
  result: TranscriptEntry | null;
  cwd?: string;
  pending?: boolean;
}

export function ToolCall({ entry, result, cwd, pending }: ToolCallProps) {
  const [expanded, setExpanded] = useState(false);

  let indicator: React.ReactNode;
  if (result) {
    indicator = result.is_error ? (
      <span className="text-red-400">&#10007;</span>
    ) : (
      <span className="text-emerald-400">&#10003;</span>
    );
  } else if (pending) {
    indicator = <Spinner className="h-3 w-3 text-gray-400" />;
  } else {
    indicator = <span className="text-emerald-400">&#10003;</span>;
  }

  const canExpand = result !== null || (entry.full_text && entry.full_text !== entry.text);
  const borderColor = result?.is_error
    ? "border-red-500/40"
    : "border-gray-300 dark:border-gray-700";

  const shorten = (text: string) => cwd ? shortenToolText(text, cwd) : text;

  return (
    <div>
      <div
        onClick={canExpand ? () => setExpanded(!expanded) : undefined}
        className={`flex items-center gap-2 px-3 py-1.5 bg-gray-200/60 dark:bg-gray-700/40 rounded-md font-mono text-xs text-gray-500 dark:text-gray-400${
          canExpand
            ? " cursor-pointer hover:bg-gray-200 dark:hover:bg-gray-700/60"
            : ""
        }`}
      >
        {indicator}
        <span className="truncate">{shorten(entry.text)}</span>
      </div>
      {expanded && (
        <div
          className={`ml-4 mt-1 border-l-2 ${borderColor} pl-3 font-mono text-xs text-gray-500 whitespace-pre-wrap break-all max-h-[400px] overflow-y-auto space-y-1`}
        >
          {entry.full_text && entry.full_text !== entry.text && (
            <div className="text-gray-600 dark:text-gray-300">
              {shorten(entry.full_text)}
            </div>
          )}
          {result && <div>{result.text}</div>}
        </div>
      )}
    </div>
  );
}
