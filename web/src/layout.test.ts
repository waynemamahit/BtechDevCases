import { setDefaultTimeout, test } from "bun:test";
import { spawnSync } from "node:child_process";
import { join } from "node:path";

setDefaultTimeout(90_000);

test("phone and desktop screens stay within the viewport", () => {
  const result = spawnSync(
    process.execPath,
    ["test", "./src/layout.browser.ts"],
    {
      cwd: join(import.meta.dir, ".."),
      env: process.env,
      encoding: "utf8",
      timeout: 90_000,
    },
  );
  const output = `${result.stdout ?? ""}\n${result.stderr ?? ""}`;
  if (result.status !== 0 || !output.includes("3 pass")) {
    throw new Error(
      output.trim() === "" ? "layout browser test failed" : output,
    );
  }
});
