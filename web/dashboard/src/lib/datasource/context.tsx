"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { DEFAULT_MODE, resolveDataSource, type DataSource, type Mode } from "./index";

interface Ctx {
  mode: Mode;
  setMode: (m: Mode) => void;
  dataSource: DataSource;
}

const DataSourceContext = createContext<Ctx | null>(null);
const STORAGE_KEY = "membrane.dashboard.mode";

export function DataSourceProvider({ children }: { children: React.ReactNode }) {
  const [mode, setModeState] = useState<Mode>(DEFAULT_MODE);

  // Per-viewer convenience only; never trusted for authorization.
  useEffect(() => {
    try {
      const saved = window.localStorage.getItem(STORAGE_KEY);
      if (saved === "live" || saved === "demo") setModeState(saved);
    } catch {
      /* storage blocked (private window) — keep default */
    }
  }, []);

  const setMode = useCallback((m: Mode) => {
    setModeState(m);
    try {
      window.localStorage.setItem(STORAGE_KEY, m);
    } catch {
      /* ignore */
    }
  }, []);

  const value = useMemo<Ctx>(
    () => ({ mode, setMode, dataSource: resolveDataSource(mode) }),
    [mode, setMode],
  );

  return <DataSourceContext.Provider value={value}>{children}</DataSourceContext.Provider>;
}

export function useDataSource(): Ctx {
  const ctx = useContext(DataSourceContext);
  if (!ctx) throw new Error("useDataSource must be used within DataSourceProvider");
  return ctx;
}
