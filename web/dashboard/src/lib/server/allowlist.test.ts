import { describe, it, expect } from "vitest";
import { checkDestination } from "./allowlist";

describe("checkDestination (BFF SSRF guard)", () => {
  const prod = { allowLoopback: false, hosts: ["assurance.agenttwin.example"] };
  const dev = { allowLoopback: true, hosts: ["assurance.agenttwin.example"] };

  it("accepts an allowlisted https host", () => {
    expect(checkDestination("https://assurance.agenttwin.example/api/v1/eval-runs", prod).ok).toBe(true);
  });
  it("rejects a host not on the allowlist", () => {
    expect(checkDestination("https://evil.example/api", prod).ok).toBe(false);
  });
  it("rejects http to a non-loopback host", () => {
    expect(checkDestination("http://assurance.agenttwin.example/api", prod).ok).toBe(false);
  });
  it("rejects loopback unless explicitly allowed", () => {
    expect(checkDestination("http://localhost:8080/api", prod).ok).toBe(false);
    expect(checkDestination("http://localhost:8080/api", dev).ok).toBe(true);
    expect(checkDestination("http://127.0.0.1:8080/api", dev).ok).toBe(true);
  });
  it("rejects embedded credentials and non-http schemes", () => {
    expect(checkDestination("https://u:p@assurance.agenttwin.example/x", prod).ok).toBe(false);
    expect(checkDestination("ftp://assurance.agenttwin.example/x", prod).ok).toBe(false);
  });
});
