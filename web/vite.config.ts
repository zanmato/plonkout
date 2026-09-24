import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";
import vueDevTools from "vite-plugin-vue-devtools";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
    tailwindcss(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.ico", "favicon.svg", "dumbbell.svg"],
      manifest: {
        name: "Plonkout",
        short_name: "Plonkout",
        description: "Workout logger",
        theme_color: "#8B5CF6",
        background_color: "#ffffff",
        display: "standalone",
        orientation: "portrait",
        scope: "/",
        start_url: "/",
        icons: [
          {
            src: "dumbbell.svg",
            sizes: "192x192",
            type: "image/svg+xml",
          },
          {
            src: "dumbbell.svg",
            sizes: "512x512",
            type: "image/svg+xml",
          },
          {
            src: "dumbbell.svg",
            sizes: "512x512",
            type: "image/svg+xml",
            purpose: "any maskable",
          },
        ],
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,ico,png,svg}"],
        // Server routes (/api, /oauth, /.well-known, /mcp) must never be answered by the service worker.
        navigateFallback: null,
        runtimeCaching: [
          {
            urlPattern: /^https:\/\/fonts\.googleapis\.com\/.*/i,
            handler: "CacheFirst",
            options: {
              cacheName: "google-fonts-cache",
              expiration: {
                maxEntries: 10,
                maxAgeSeconds: 60 * 60 * 24 * 365, // <== 365 days
              },
            },
          },
        ],
      },
    }),
  ],
  server: {
    // The Go server answers everything that is not the SPA.
    proxy: Object.fromEntries(
      ["/api", "/oauth", "/.well-known", "/mcp"].map((path) => [
        path,
        { target: "http://127.0.0.1:8090", changeOrigin: false },
      ]),
    ),
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["src/tests/setup.ts"],
  },
});
