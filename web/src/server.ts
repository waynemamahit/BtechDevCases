import { join } from "node:path";

export type WebConfig = {
  apiBaseUrl: string;
  port: number;
};

export function loadConfig(env: {
  API_BASE_URL?: string;
  WEB_PORT?: string;
}): WebConfig {
  const apiBaseUrl = env.API_BASE_URL ?? "";
  if (apiBaseUrl === "") {
    throw new Error("API_BASE_URL is required");
  }

  const rawPort = env.WEB_PORT ?? "";
  const port = rawPort === "" ? 5173 : Number(rawPort);
  if (!Number.isInteger(port) || port <= 0 || port > 65535) {
    throw new Error("WEB_PORT must be a positive integer");
  }

  return { apiBaseUrl, port };
}

export async function handleRequest(
  request: Request,
  options: { apiBaseUrl: string; distDir: string },
): Promise<Response> {
  const url = new URL(request.url);
  if (url.pathname === "/config.json") {
    return Response.json({ apiBaseUrl: options.apiBaseUrl });
  }

  const filePath = resolveDistFile(options.distDir, url.pathname);
  if (filePath) {
    const file = Bun.file(filePath);
    if (await file.exists()) {
      return new Response(file);
    }
  }

  const indexPath = join(options.distDir, "index.html");
  const index = Bun.file(indexPath);
  if (await index.exists()) {
    return new Response(index);
  }
  return new Response("not found", { status: 404 });
}

function resolveDistFile(distDir: string, pathname: string): string | null {
  const relative =
    pathname === "/" ? "index.html" : pathname.replace(/^\/+/, "");
  const parts = relative.split("/");
  if (
    parts.some(
      (part) =>
        part === "" || part === "." || part === ".." || part.includes("\0"),
    )
  ) {
    return null;
  }
  return join(distDir, ...parts);
}

if (import.meta.main) {
  let config: WebConfig;
  try {
    config = loadConfig({
      API_BASE_URL: Bun.env.API_BASE_URL,
      WEB_PORT: Bun.env.WEB_PORT,
    });
  } catch (error) {
    console.error(error instanceof Error ? error.message : error);
    process.exit(1);
  }

  const distDir = join(import.meta.dir, "..", "dist");
  Bun.serve({
    hostname: "0.0.0.0",
    port: config.port,
    fetch(request) {
      return handleRequest(request, { apiBaseUrl: config.apiBaseUrl, distDir });
    },
  });
}
