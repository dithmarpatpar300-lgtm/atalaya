import type { NextConfig } from "next";

const esDesarrollo = process.env.NODE_ENV === "development";
const apiDesarrollo = process.env.ATALAYA_API ?? "http://127.0.0.1:7420";

// En producción, exportación estática que el binario Go sirve embebida (sin servidor Node); en desarrollo, `next dev`
// reenvía /api al backend local (otro con ATALAYA_API).
const nextConfig: NextConfig = {
  ...(esDesarrollo
    ? {
        async rewrites() {
          return [{ source: "/api/:ruta*", destination: `${apiDesarrollo}/api/:ruta*` }];
        },
      }
    : { output: "export" as const }),
  images: { unoptimized: true },
  poweredByHeader: false,
  turbopack: {
    rules: {
      "*.css": {
        condition: { not: { path: /\.module\.css$/ } },
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
