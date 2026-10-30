import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Disconnected, Planned, ErrorState, Empty } from "./states";

describe("data-state views", () => {
  it("Disconnected renders an honest not-connected message", () => {
    render(<Disconnected detail="backend not reachable (timeout)" />);
    expect(screen.getByTestId("state-disconnected")).toBeInTheDocument();
    expect(screen.getByText(/backend not reachable/i)).toBeInTheDocument();
  });

  it("Planned marks a capability as not-yet-implemented with its phase", () => {
    render(<Planned phase="D2" />);
    expect(screen.getByTestId("state-planned")).toHaveTextContent(/Planned — D2/);
  });

  it("ErrorState uses an alert role", () => {
    render(<ErrorState detail="boom" />);
    expect(screen.getByRole("alert")).toHaveTextContent("boom");
  });

  it("Empty renders a hint", () => {
    render(<Empty hint="no scans yet" />);
    expect(screen.getByTestId("state-empty")).toHaveTextContent("no scans yet");
  });
});
