import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: path.resolve("../internal/server/web/dist"),
    emptyOutDir: true
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:7777"
    }
  }
});
