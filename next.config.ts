import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  htmlLimitedBots: /.*/,
  images: { unoptimized: true },
};

export default nextConfig;
