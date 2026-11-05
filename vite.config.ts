import { defineConfig } from "vite";

export default defineConfig({
    build: {
        outDir: "dist",
        rollupOptions: {
            output: {
                manualChunks(id) {
                    if (id.includes("xlsx-js-style")) {
                        return "xlsx";
                    }
                    return undefined;
                }
            }
        }
    }
});