import { afterAll, beforeAll, expect, setDefaultTimeout, test } from "bun:test";
import { join } from "node:path";
import { GlobalRegistrator } from "@happy-dom/global-registrator";

setDefaultTimeout(60_000);

const phone = { name: "narrow phone", width: 320, height: 800 };
const desktop = { name: "desktop", width: 1280, height: 800 };
const specEmail = "ada@example.com";
const longEmail =
  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@example.com";

type Browser = import("playwright").Browser;
type Page = import("playwright").Page;
type Locator = import("playwright").Locator;
type ViteDevServer = import("vite").ViteDevServer;

let browser: Browser | undefined;
let server: ViteDevServer | undefined;
let baseURL = "";

function tokenExpiringAt(expSeconds: number): string {
  const payload = btoa(JSON.stringify({ exp: expSeconds }))
    .replaceAll("+", "-")
    .replaceAll("/", "_")
    .replace(/=+$/g, "");
  return `eyJhbGciOiJIUzI1NiJ9.${payload}.sig`;
}

beforeAll(async () => {
  await GlobalRegistrator.unregister();
  const { createServer } = await import("vite");
  const { chromium } = await import("playwright");
  server = await createServer({
    configFile: join(import.meta.dir, "..", "vite.config.ts"),
    server: { host: "127.0.0.1", port: 4173, strictPort: false },
    plugins: [
      {
        name: "layout-test-config",
        configureServer(devServer) {
          devServer.middlewares.use((req, res, next) => {
            const path = req.url?.split("?")[0];
            if (path !== "/config.json") {
              next();
              return;
            }
            const host = req.headers.host ?? "127.0.0.1:4173";
            res.statusCode = 200;
            res.setHeader("content-type", "application/json");
            res.end(JSON.stringify({ apiBaseUrl: `http://${host}` }));
          });
        },
      },
    ],
  });
  await server.listen();
  baseURL = server.resolvedUrls?.local[0] ?? "";
  if (baseURL === "") {
    throw new Error("vite did not report a local url");
  }
  browser = await chromium.launch();
}, 60_000);

afterAll(async () => {
  await browser?.close();
  await server?.close();
});

async function openPage(
  viewport: { width: number; height: number },
  email: string,
): Promise<Page> {
  if (!browser) {
    throw new Error("browser is not open");
  }
  const page = await browser.newPage({ viewport });
  const token = tokenExpiringAt(Math.floor(Date.now() / 1000) + 60 * 60);
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/login") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ token }),
      });
      return;
    }
    if (url.pathname === "/me") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ id: "user-1", email, token }),
      });
      return;
    }
    await route.continue();
  });
  await page.goto(baseURL);
  await page.getByRole("button", { name: "Log in" }).waitFor();
  return page;
}

async function assertNoHorizontalPageScroll(page: Page) {
  const metrics = await page.evaluate(() => {
    const root = document.documentElement;
    const body = document.body;
    return {
      rootScroll: root.scrollWidth,
      rootClient: root.clientWidth,
      bodyScroll: body.scrollWidth,
      bodyClient: body.clientWidth,
    };
  });
  expect(metrics.rootScroll).toBeLessThanOrEqual(metrics.rootClient);
  expect(metrics.bodyScroll).toBeLessThanOrEqual(metrics.bodyClient);
}

async function assertWithinViewport(page: Page, locator: Locator) {
  const viewport = page.viewportSize();
  if (!viewport) {
    throw new Error("viewport is missing");
  }
  const box = await locator.boundingBox();
  expect(box).not.toBeNull();
  if (!box) {
    return;
  }
  expect(box.width).toBeGreaterThan(0);
  expect(box.height).toBeGreaterThan(0);
  expect(box.x).toBeGreaterThanOrEqual(-1);
  expect(box.x + box.width).toBeLessThanOrEqual(viewport.width + 1);
  expect(box.y).toBeGreaterThanOrEqual(-1);
  expect(box.y + box.height).toBeLessThanOrEqual(viewport.height + 1);
}

async function assertWidthStep(
  page: Page,
  viewportWidth: number,
  selector: string,
) {
  const locator = page.locator(selector);
  const box = await locator.boundingBox();
  expect(box).not.toBeNull();
  if (!box) {
    return;
  }
  const maxWidth = await locator.evaluate(
    (element) => getComputedStyle(element).maxWidth,
  );
  if (viewportWidth >= 768) {
    expect(maxWidth === "32rem" || maxWidth === "512px").toBe(true);
    expect(box.width).toBeGreaterThan(448);
    expect(box.width).toBeLessThanOrEqual(512.5);
  } else {
    expect(maxWidth === "24rem" || maxWidth === "384px").toBe(true);
    expect(box.width).toBeLessThanOrEqual(viewportWidth);
    expect(box.width).toBeGreaterThan(200);
  }
}

async function assertAuthFits(
  page: Page,
  viewport: { width: number; height: number },
  panel: "login" | "register",
) {
  await assertNoHorizontalPageScroll(page);
  await assertWithinViewport(page, page.getByRole("radio", { name: "Log in" }));
  await assertWithinViewport(
    page,
    page.getByRole("radio", { name: "Register" }),
  );
  await assertWithinViewport(page, page.getByLabel("Email"));
  await assertWithinViewport(
    page,
    page.getByLabel("Password", { exact: true }),
  );
  if (panel === "login") {
    await assertWithinViewport(
      page,
      page.getByRole("button", { name: "Log in" }),
    );
    expect(await page.getByLabel("Confirm password").count()).toBe(0);
  } else {
    await assertWithinViewport(page, page.getByLabel("Confirm password"));
    await assertWithinViewport(
      page,
      page.getByRole("button", { name: "Create account" }),
    );
    expect(await page.getByRole("button", { name: "Log in" }).count()).toBe(0);
  }
  await assertWidthStep(page, viewport.width, "main .grid");
  await assertNoHorizontalPageScroll(page);
}

async function assertWelcomeFits(
  page: Page,
  viewport: { width: number; height: number },
  email: string,
) {
  const greeting = page.getByText(`Hello ${email}, welcome back`, {
    exact: true,
  });
  await greeting.waitFor();
  expect(await greeting.textContent()).toBe(`Hello ${email}, welcome back`);
  await assertNoHorizontalPageScroll(page);
  await assertWithinViewport(page, greeting);
  await assertWidthStep(page, viewport.width, "main .card");
  await assertNoHorizontalPageScroll(page);
}

async function signIn(page: Page, email: string) {
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("s3cret");
  await page.getByRole("button", { name: "Log in" }).click();
  await page
    .getByText(`Hello ${email}, welcome back`, { exact: true })
    .waitFor();
}

for (const viewport of [phone, desktop]) {
  test(`${viewport.name} keeps login, register, and welcome on screen`, async () => {
    const page = await openPage(viewport, specEmail);
    try {
      await assertAuthFits(page, viewport, "login");
      await page.getByRole("radio", { name: "Register" }).click();
      await page.getByRole("button", { name: "Create account" }).waitFor();
      await assertAuthFits(page, viewport, "register");
      await page.getByRole("radio", { name: "Log in" }).click();
      await page.getByRole("button", { name: "Log in" }).waitFor();
      await signIn(page, specEmail);
      await assertWelcomeFits(page, viewport, specEmail);
    } finally {
      await page.close();
    }
  });
}

test("narrow phone keeps a long welcome sentence on screen", async () => {
  const page = await openPage(phone, longEmail);
  try {
    await signIn(page, longEmail);
    await assertWelcomeFits(page, phone, longEmail);
  } finally {
    await page.close();
  }
});
