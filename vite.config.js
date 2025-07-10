import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";
import vueDevTools from "vite-plugin-vue-devtools";
import { resolve } from "path";

export default defineConfig(({ _command, mode }) => {
  const isProduction = mode === 'production'
  const base = isProduction && process.env.GITHUB_PAGES ? '/plonkout/' : '/'
  const pwaScope = isProduction && process.env.GITHUB_PAGES ? '/plonkout/' : '/'
  const pwaStartUrl = isProduction && process.env.GITHUB_PAGES ? '/plonkout/' : '/'
  
  return {
  base,
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
        scope: pwaScope,
        start_url: pwaStartUrl,
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
        navigateFallback: null, // Disable navigate fallback for GitHub Pages
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
  resolve: {
    alias: {
      "@": resolve(__dirname, "src"),
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["src/tests/setup.js"],
  },
  }
});
