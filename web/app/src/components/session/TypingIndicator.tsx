export function TypingIndicator({ detail }: { detail?: string }) {
  return (
    <div className="flex justify-start">
      <div className="bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-500 text-sm px-3 py-2 rounded-xl rounded-bl-sm flex items-center gap-2">
        <span className="flex gap-1">
          <span className="typing-dot w-1.5 h-1.5 rounded-full bg-current" />
          <span
            className="typing-dot w-1.5 h-1.5 rounded-full bg-current"
            style={{ animationDelay: "0.2s" }}
          />
          <span
            className="typing-dot w-1.5 h-1.5 rounded-full bg-current"
            style={{ animationDelay: "0.4s" }}
          />
        </span>
        {detail && <span className="text-xs truncate">{detail}</span>}
      </div>
    </div>
  );
}
