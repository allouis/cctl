import { useMemo } from "react";
import { marked } from "marked";
import DOMPurify from "dompurify";

marked.setOptions({ breaks: true, gfm: true });

export function AssistantBubble({ text }: { text: string }) {
  const html = useMemo(
    () => DOMPurify.sanitize(marked.parse(text) as string),
    [text],
  );

  return (
    <div className="flex justify-start">
      <div
        className="max-w-[85%] bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-200 text-sm px-3 py-2 rounded-xl rounded-bl-sm markdown"
        dangerouslySetInnerHTML={{ __html: html }}
      />
    </div>
  );
}
