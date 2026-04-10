import { useState, useEffect, useRef } from "react";
import { Modal } from "./Modal";
import { getSystemPrompt, saveSystemPrompt } from "../../api/client";

interface SystemPromptModalProps {
  open: boolean;
  onClose: () => void;
}

export function SystemPromptModal({ open, onClose }: SystemPromptModalProps) {
  const [content, setContent] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (open) {
      setError(null);
      setSaving(false);
      setLoaded(false);
      getSystemPrompt()
        .then((res) => {
          setContent(res.content);
          setLoaded(true);
          setTimeout(() => textareaRef.current?.focus(), 0);
        })
        .catch((e) => setError(e instanceof Error ? e.message : "Load failed"));
    }
  }, [open]);

  async function handleSave() {
    setSaving(true);
    setError(null);
    try {
      await saveSystemPrompt(content);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
      setSaving(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose}>
      <div className="relative bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-xl shadow-xl w-full max-w-lg">
        <div className="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
            System Prompt
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Appended to every new and resumed session.
          </p>
        </div>
        <div className="px-5 py-4">
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            disabled={!loaded}
            placeholder="Enter instructions for all sessions..."
            rows={12}
            className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-2 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 placeholder-gray-400 dark:placeholder-gray-600 resize-y font-mono"
          />
          {error && <div className="mt-2 text-xs text-red-500">{error}</div>}
        </div>
        <div className="px-5 py-3 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={saving || !loaded}
            className="px-3 py-1.5 text-sm font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-md transition-colors disabled:opacity-50"
          >
            {saving ? "Saving..." : "Save"}
          </button>
        </div>
      </div>
    </Modal>
  );
}
