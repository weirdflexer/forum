import { useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import { message } from "../lib/errors";

type LoadState<T> = {
  path: string | null;
  data: T | null;
  error: string;
  loading: boolean;
};

// A null path disables loading; old requests cannot update a different page.
export function useLoad<T>(path: string | null, version = 0) {
  const [state, setState] = useState<LoadState<T>>({
    path: null,
    data: null,
    error: "",
    loading: false,
  });
  const [retry, setRetry] = useState(0);
  const reload = useCallback(() => setRetry((retry) => retry + 1), []);
  useEffect(() => {
    if (path === null) return;
    const controller = new AbortController();
    setState((previous) => ({
      path,
      data: previous.path === path ? previous.data : null,
      error: "",
      loading: true,
    }));
    api<T>(path, { signal: controller.signal })
      .then((data) => {
        if (!controller.signal.aborted)
          setState({ path, data, error: "", loading: false });
      })
      .catch((error) => {
        if (!controller.signal.aborted)
          setState((previous) => ({
            ...previous,
            error: message(error),
            loading: false,
          }));
      });
    return () => controller.abort();
  }, [path, version, retry]);
  const current = path !== null && path === state.path;
  return {
    data: current ? state.data : null,
    error: current ? state.error : "",
    loading: path !== null && (!current || state.loading),
    reload,
  };
}
