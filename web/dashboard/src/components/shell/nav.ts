// The nine product sections (DASHBOARD_AND_NETLIFY_WORKFLOW.md §2). Each route
// renders a real view or an honest Planned/empty state — never a dead link.
export interface NavItem {
  href: string;
  label: string;
  group: "Governance" | "Assurance" | "Platform";
}

export const NAV: NavItem[] = [
  { href: "/", label: "Overview", group: "Governance" },
  { href: "/code-governance", label: "Code Governance", group: "Governance" },
  { href: "/security", label: "Security Center", group: "Governance" },
  { href: "/agents", label: "Agent Twin Lab", group: "Assurance" },
  { href: "/simulations", label: "Simulation Center", group: "Assurance" },
  { href: "/traces", label: "Trace Explorer", group: "Assurance" },
  { href: "/release-assurance", label: "Release Assurance", group: "Assurance" },
  { href: "/reports", label: "Reports", group: "Platform" },
  { href: "/settings", label: "Settings", group: "Platform" },
];
