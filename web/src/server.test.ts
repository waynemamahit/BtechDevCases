import { expect, test } from "bun:test";
import { join } from "node:path";
import { handleRequest, loadConfig } from "./server";

test("GET /config.json returns apiBaseUrl", async () => {
  const response = await handleRequest(
    new Request("http://localhost/config.json"),
    {
      apiBaseUrl: "http://localhost:8080",
      distDir: join(import.meta.dir, "missing-dist"),
    },
  );

  expect(response.status).toBe(200);
  expect(await response.json()).toEqual({
    apiBaseUrl: "http://localhost:8080",
  });
});

test("missing API_BASE_URL fails", () => {
  expect(() => loadConfig({})).toThrow(/API_BASE_URL/);
  expect(() => loadConfig({ API_BASE_URL: "" })).toThrow(/API_BASE_URL/);
});

test("WEB_PORT defaults to 5173", () => {
  expect(loadConfig({ API_BASE_URL: "http://localhost:8080" }).port).toBe(5173);
});

test("process exits when API_BASE_URL is missing", () => {
  const result = Bun.spawnSync({
    cmd: [process.execPath, "src/server.ts"],
    cwd: join(import.meta.dir, ".."),
    env: {
      PATH: process.env.PATH ?? "",
      Path: process.env.Path ?? process.env.PATH ?? "",
      SYSTEMROOT: process.env.SYSTEMROOT ?? "",
    },
    stdout: "pipe",
    stderr: "pipe",
  });

  const stderr = result.stderr.toString();
  expect(result.exitCode).not.toBe(0);
  expect(stderr).toContain("API_BASE_URL");
});
