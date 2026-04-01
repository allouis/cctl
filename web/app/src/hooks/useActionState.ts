import { useState, useCallback } from "react";

interface ActionState {
  loading: boolean;
  error: string | null;
}

export function useActionState() {
  const [state, setState] = useState<ActionState>({
    loading: false,
    error: null,
  });

  const execute = useCallback(
    async (fn: () => Promise<unknown>, opts?: { stayLoading?: boolean }) => {
      setState({ loading: true, error: null });
      try {
        await fn();
        if (!opts?.stayLoading) {
          setState({ loading: false, error: null });
        }
      } catch (e) {
        setState({
          loading: false,
          error: e instanceof Error ? e.message : "Unknown error",
        });
      }
    },
    [],
  );

  const reset = useCallback(() => {
    setState({ loading: false, error: null });
  }, []);

  return { ...state, execute, reset };
}
