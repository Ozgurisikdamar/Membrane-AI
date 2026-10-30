"use client";

import type { Sourced } from "@/lib/datasource/types";
import { Loading, ErrorState, Disconnected } from "./states";

type Load<T> =
  | { status: "loading" }
  | { status: "ready"; data: Sourced<T> }
  | { status: "error"; error: string };

/**
 * Renders one of the honest data-states, or the children with the resolved
 * Sourced value. Keeps every data-backed view consistent: loading → hard error
 * → backend-unavailable → content.
 */
export function SourcedView<T>({
  load,
  loadingLabel,
  children,
}: {
  load: Load<T>;
  loadingLabel?: string;
  children: (data: Sourced<T>) => React.ReactNode;
}) {
  if (load.status === "loading") return <Loading label={loadingLabel} />;
  if (load.status === "error") return <ErrorState detail={load.error} />;
  if (load.data.source === "unavailable") return <Disconnected detail={load.data.note} />;
  return <>{children(load.data)}</>;
}
