import { defineConfig } from "vitest/config";
import * as path from "path";

export default defineConfig({
  test: {
    root: "src",
    include: ["__tests__/**/*.test.ts"],
  },
  resolve: {
    alias: {
      vscode: path.resolve(__dirname, "src/__tests__/__mocks__/vscode.ts"),
    },
  },
});
