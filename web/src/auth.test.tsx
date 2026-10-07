import { afterEach, expect, jest, mock, test } from "bun:test";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { App } from "./App";

const originalFetch = globalThis.fetch;
const welcome = "Hello ada@example.com, welcome back";

afterEach(() => {
  cleanup();
  sessionStorage.clear();
  globalThis.fetch = originalFetch;
  if (jest.isFakeTimers()) {
    jest.clearAllTimers();
    jest.useRealTimers();
  }
});

function tokenExpiringAt(expSeconds: number): string {
  const payload = btoa(JSON.stringify({ exp: expSeconds }))
    .replaceAll("+", "-")
    .replaceAll("/", "_")
    .replace(/=+$/g, "");
  return `eyJhbGciOiJIUzI1NiJ9.${payload}.sig`;
}

function jsonResponse(body: unknown, status = 200): Response {
  return Response.json(body, { status });
}

async function settle() {
  for (let i = 0; i < 20; i++) {
    await act(async () => {
      await Promise.resolve();
    });
  }
}

test("the page shows one auth form at a time", async () => {
  globalThis.fetch = mock(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.endsWith("/config.json")) {
      return Response.json({ apiBaseUrl: "http://localhost:8080" });
    }
    throw new Error(`unexpected fetch ${url}`);
  }) as unknown as typeof fetch;

  render(<App />);

  expect(await screen.findByRole("button", { name: "Log in" })).toBeTruthy();
  expect(screen.queryByLabelText("Confirm password")).toBeNull();

  fireEvent.click(screen.getByRole("radio", { name: "Register" }));

  expect(
    await screen.findByRole("button", { name: "Create account" }),
  ).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Log in" })).toBeNull();
  expect(screen.getByLabelText("Confirm password")).toBeTruthy();
});

test("accepted registration returns to login and stores no token", async () => {
  globalThis.fetch = mock(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.endsWith("/config.json")) {
      return jsonResponse({ apiBaseUrl: "http://localhost:8080" });
    }
    if (url.endsWith("/register")) {
      return jsonResponse({ id: "user-1", email: "ada@example.com" }, 201);
    }
    throw new Error(`unexpected fetch ${url}`);
  }) as unknown as typeof fetch;

  render(<App />);
  fireEvent.click(await screen.findByRole("radio", { name: "Register" }));
  fireEvent.change(screen.getByLabelText("Email"), {
    target: { value: "ada@example.com" },
  });
  fireEvent.change(screen.getByLabelText("Password"), {
    target: { value: "s3cret" },
  });
  fireEvent.change(screen.getByLabelText("Confirm password"), {
    target: { value: "s3cret" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Create account" }));

  expect(await screen.findByRole("button", { name: "Log in" })).toBeTruthy();
  expect(screen.queryByLabelText("Confirm password")).toBeNull();
  expect(screen.queryByText(welcome)).toBeNull();
  expect(sessionStorage.getItem("token")).toBeNull();
});

test("the protected screen ends the session at the token exp", async () => {
  const now = Date.parse("2026-10-07T12:00:00.000Z");
  const exp = Math.floor(now / 1000) + 15 * 60;
  const token = tokenExpiringAt(exp);

  globalThis.fetch = mock(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.endsWith("/config.json")) {
      return jsonResponse({ apiBaseUrl: "http://localhost:8080" });
    }
    if (url.endsWith("/login")) {
      return jsonResponse({ token });
    }
    if (url.endsWith("/me")) {
      return jsonResponse({
        id: "user-1",
        email: "ada@example.com",
        token,
      });
    }
    throw new Error(`unexpected fetch ${url}`);
  }) as unknown as typeof fetch;

  render(<App />);
  expect(await screen.findByRole("button", { name: "Log in" })).toBeTruthy();

  jest.useFakeTimers({ now });
  fireEvent.change(screen.getByLabelText("Email"), {
    target: { value: "ada@example.com" },
  });
  fireEvent.change(screen.getByLabelText("Password"), {
    target: { value: "s3cret" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Log in" }));
  await settle();

  expect(screen.getByText(welcome).textContent).toBe(welcome);
  expect(sessionStorage.getItem("token")).toBe(token);

  await act(async () => {
    jest.advanceTimersByTime(14 * 60 * 1000);
  });
  expect(screen.getByText(welcome).textContent).toBe(welcome);
  expect(sessionStorage.getItem("token")).toBe(token);

  await act(async () => {
    jest.advanceTimersByTime(60 * 1000);
  });
  expect(screen.queryByText(welcome)).toBeNull();
  expect(screen.getByRole("button", { name: "Log in" })).toBeTruthy();
  expect(sessionStorage.getItem("token")).toBeNull();
});

test("a 401 from GET /me removes the stored token and leaves the protected screen", async () => {
  const token = tokenExpiringAt(Math.floor(Date.now() / 1000) + 60 * 60);
  let identityCalls = 0;

  globalThis.fetch = mock(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.endsWith("/config.json")) {
      return jsonResponse({ apiBaseUrl: "http://localhost:8080" });
    }
    if (url.endsWith("/login")) {
      return jsonResponse({ token });
    }
    if (url.endsWith("/me")) {
      identityCalls += 1;
      if (identityCalls === 1) {
        return jsonResponse({
          id: "user-1",
          email: "ada@example.com",
          token,
        });
      }
      return jsonResponse({ error: "unauthorized" }, 401);
    }
    throw new Error(`unexpected fetch ${url}`);
  }) as unknown as typeof fetch;

  const first = render(<App />);
  expect(await screen.findByRole("button", { name: "Log in" })).toBeTruthy();
  fireEvent.change(screen.getByLabelText("Email"), {
    target: { value: "ada@example.com" },
  });
  fireEvent.change(screen.getByLabelText("Password"), {
    target: { value: "s3cret" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Log in" }));

  expect(await screen.findByText(welcome)).toBeTruthy();
  expect(screen.getByText(welcome).textContent).toBe(welcome);
  expect(sessionStorage.getItem("token")).toBe(token);

  first.unmount();
  render(<App />);

  expect(await screen.findByRole("button", { name: "Log in" })).toBeTruthy();
  expect(screen.queryByText(welcome)).toBeNull();
  expect(sessionStorage.getItem("token")).toBeNull();
});

test("a 401 from GET /me after login stores no token", async () => {
  const token = tokenExpiringAt(Math.floor(Date.now() / 1000) + 60 * 60);

  globalThis.fetch = mock(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.endsWith("/config.json")) {
      return jsonResponse({ apiBaseUrl: "http://localhost:8080" });
    }
    if (url.endsWith("/login")) {
      return jsonResponse({ token });
    }
    if (url.endsWith("/me")) {
      return jsonResponse({ error: "unauthorized" }, 401);
    }
    throw new Error(`unexpected fetch ${url}`);
  }) as unknown as typeof fetch;

  render(<App />);
  expect(await screen.findByRole("button", { name: "Log in" })).toBeTruthy();
  fireEvent.change(screen.getByLabelText("Email"), {
    target: { value: "ada@example.com" },
  });
  fireEvent.change(screen.getByLabelText("Password"), {
    target: { value: "s3cret" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Log in" }));

  expect(await screen.findByText("session rejected")).toBeTruthy();
  expect(screen.queryByText(welcome)).toBeNull();
  expect(screen.getByRole("button", { name: "Log in" })).toBeTruthy();
  expect(sessionStorage.getItem("token")).toBeNull();
});
