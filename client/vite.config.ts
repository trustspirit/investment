import { defineConfig } from "vite";
import type { Plugin } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// Vite's SPA fallback skips URLs containing dots (treats them as file requests).
// Korean stock symbols like /stock/005930.KS need to be served as SPA routes.
function spaRoutesFix(): Plugin {
  return {
    name: "spa-routes-fix",
    configureServer(server) {
      server.middlewares.use((req, _res, next) => {
        if (req.url?.startsWith("/stock/")) {
          req.url = "/";
        }
        next();
      });
    },
  };
}

export default defineConfig({
  plugins: [spaRoutesFix(), react(), tailwindcss()],
  server: {
    proxy: {
      "/api": { target: "http://localhost:8081", changeOrigin: true },
      "/ws": { target: "ws://localhost:8081", ws: true },
    },
  },
});
