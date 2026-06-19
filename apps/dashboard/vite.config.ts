import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // Durante desarrollo, /api se redirige al control-plane en Go
      "/api": {
        // In Docker the control-plane is reachable at its service name.
        // On bare host, set VITE_API_URL=http://localhost:3000.
        target: process.env.VITE_API_URL ?? "http://control-plane:3000",
        changeOrigin: true,
      },
    },
  },
});
