import type { Config } from "tailwindcss";

// Membrane AI brand tokens, reused from docs/landing/index.html and the CLI
// report (clients/cli/internal/report/html.go): teal accent, warm paper ground.
const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        accent: {
          DEFAULT: "#0f766e",
          strong: "#0d9488",
          deep: "#0e7490",
        },
        paper: "#f6f3ec",
        ink: "#17212e",
        muted: "#5b6b7a",
        // Source/provenance palette.
        live: "#0d9488",
        demo: "#b45309",
        unavailable: "#94a3b8",
        // Gate decisions.
        pass: "#15803d",
        warn: "#b45309",
        block: "#b91c1c",
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "ui-monospace", "SFMono-Regular", "monospace"],
      },
    },
  },
  plugins: [],
};

export default config;
