import type { Metadata } from "next";
import "./globals.css";
import { DataSourceProvider } from "@/lib/datasource/context";
import { AppShell } from "@/components/shell/app-shell";

export const metadata: Metadata = {
  title: "MEMBRANE.AI — Governance & Agent Assurance",
  description:
    "Unified Membrane AI dashboard: Code Governance and Agent Assurance evidence, evaluations, simulations and release decisions.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <DataSourceProvider>
          <AppShell>{children}</AppShell>
        </DataSourceProvider>
      </body>
    </html>
  );
}
