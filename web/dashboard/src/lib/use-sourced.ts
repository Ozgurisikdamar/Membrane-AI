"use client";

import { useEffect, useState } from "react";
import type { Sourced } from "./datasource/types";

type Load<T> = { status: "loading" } | { status: "ready"; data: Sourced<T> } | { status: "error"; error: string };

/**
 * Loads a Sourced<T> from an async data-source call, re-running when `deps`
 * change (e.g. the live/demo mode). Surfaces loading and hard-error states; a
 * backend that is merely unreachable resolves to a `ready` Sourced value whose
 * source is "unavailable", so the caller renders a Disconnected state, not a crash.
 */
export function useSourced<T>(fn: () => Promise<Sourced<T>>, deps: unknown[]): Load<T> {
  const [state, setState] = useState<Load<T>>({ status: "loading" });
  useEffect(() => {
    let alive = true;
    setState({ status: "loading" });
    fn()
      .then((data) => alive && setState({ status: "ready", data }))
      .catch((e) => alive && setState({ status: "error", error: e instanceof Error ? e.message : String(e) }));
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return state;
}
