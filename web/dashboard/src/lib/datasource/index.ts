import type { DataSource, Provenance } from "./types";
import { DemoDataSource } from "./demo";
import { LiveDataSource } from "./live";

export type Mode = "demo" | "live";

/** Demo is the default: a static Netlify deploy is honest out of the box. */
export const DEFAULT_MODE: Mode = "demo";

const demoSource = new DemoDataSource();
const liveSource = new LiveDataSource();

export function resolveDataSource(mode: Mode): DataSource {
  return mode === "live" ? liveSource : demoSource;
}

export type { DataSource, Provenance };
export * from "./types";
