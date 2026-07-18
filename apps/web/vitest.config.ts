import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

export default defineConfig({
  oxc: {
    jsx: {
      runtime: "automatic",
      importSource: "react"
    }
  },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url))
    }
  },
  test: {
    environment: "node",
    setupFiles: ["./test/setup.ts"],
    include: ["tests/**/*.test.ts", "tests/**/*.test.tsx"],
    restoreMocks: true,
    clearMocks: true,
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary", "html", "lcov"],
      reportsDirectory: "./coverage",
      include: [
        "src/lib/**/*.ts",
        "src/hooks/**/*.ts",
        "src/components/**/*.tsx",
        "src/features/**/*.ts",
        "src/features/**/*.tsx"
      ],
      exclude: ["**/*.d.ts", "**/index.ts", "**/*.test.*"]
    }
  }
});
