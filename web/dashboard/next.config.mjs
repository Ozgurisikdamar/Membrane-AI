/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  // Type-checking stays on during build (fail-fast on type errors). ESLint is
  // run as a separate `npm run lint` gate, not during build, to keep the
  // production build deterministic.
  eslint: { ignoreDuringBuilds: true },
  // Security headers applied to every response (tightened further per-request
  // by middleware CSP in a later phase).
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
        ],
      },
    ];
  },
};

export default nextConfig;
