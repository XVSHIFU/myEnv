import type { NextConfig } from 'next';

// Export routes at the root; repository prefixes belong to static links/assets,
// not the prerender request router (the pinned exporter cannot resolve basePath).
const nextConfig: NextConfig = {output:'export',assetPrefix:process.env.NEXT_PUBLIC_BASE_PATH||''};

export default nextConfig;
