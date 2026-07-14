import { defineConfig } from "vitest/config";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";

const AUTH_BASE = process.env.VITE_AUTH_BASE_URL ?? "http://localhost:8080";
const API_BASE = process.env.VITE_API_BASE_URL ?? "http://localhost:8081";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      // Auth service: /v1/auth/*, /v1/users/*, /healthz, /readyz
      "/v1": {
        target: AUTH_BASE,
        changeOrigin: true,
      },
      // Enrollment service: /api/v1/*, /healthz (enrollment)
      "/api": {
        target: API_BASE,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "node",
  },
});
