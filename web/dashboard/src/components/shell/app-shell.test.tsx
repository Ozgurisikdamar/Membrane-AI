import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AppShell } from "./app-shell";
import { DataSourceProvider } from "@/lib/datasource/context";

// next/navigation + next/link stubs for jsdom.
vi.mock("next/navigation", () => ({ usePathname: () => "/" }));
vi.mock("next/link", () => ({
  default: ({ href, children, ...rest }: { href: string; children: React.ReactNode }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}));

function renderShell() {
  return render(
    <DataSourceProvider>
      <AppShell>
        <p>content</p>
      </AppShell>
    </DataSourceProvider>,
  );
}

describe("AppShell", () => {
  beforeEach(() => {
    try {
      window.localStorage.clear();
    } catch {
      /* ignore */
    }
  });

  it("renders all nine product sections", () => {
    renderShell();
    for (const label of [
      "Overview",
      "Code Governance",
      "Security Center",
      "Agent Twin Lab",
      "Simulation Center",
      "Trace Explorer",
      "Release Assurance",
      "Reports",
      "Settings",
    ]) {
      expect(screen.getByRole("link", { name: label })).toBeInTheDocument();
    }
  });

  it("shows the DEMO DATA banner in the default (demo) mode", () => {
    renderShell();
    expect(screen.getByTestId("demo-banner")).toHaveTextContent(/DEMO DATA/);
  });

  it("hides the demo banner after switching to Live mode", async () => {
    const user = userEvent.setup();
    renderShell();
    expect(screen.getByTestId("demo-banner")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Live" }));
    expect(screen.queryByTestId("demo-banner")).not.toBeInTheDocument();
  });
});
