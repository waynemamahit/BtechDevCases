import { type FormEvent, useEffect, useRef, useState } from "react";
import { Welcome } from "./Welcome";

const TOKEN_KEY = "token";

type TimerHandle = { id: number | null };
type Notice = { tone: "error" | "success"; text: string };
type Panel = "login" | "register";

function tokenExpiryMs(token: string): number {
  const segment = token.split(".")[1];
  if (!segment) {
    throw new Error("token is missing a payload");
  }
  const padded = segment
    .replace(/-/g, "+")
    .replace(/_/g, "/")
    .padEnd(segment.length + ((4 - (segment.length % 4)) % 4), "=");
  const payload = JSON.parse(atob(padded)) as { exp?: number };
  if (typeof payload.exp !== "number") {
    throw new Error("token is missing exp");
  }
  return payload.exp * 1000;
}

function rememberToken(
  token: string,
  timer: TimerHandle,
  onExpire: () => void,
) {
  sessionStorage.setItem(TOKEN_KEY, token);
  if (timer.id != null) {
    window.clearTimeout(timer.id);
  }
  const id = window.setTimeout(
    onExpire,
    Math.max(0, tokenExpiryMs(token) - Date.now()),
  );
  timer.id = id;
}

function forgetToken(timer: TimerHandle) {
  if (timer.id != null) {
    window.clearTimeout(timer.id);
    timer.id = null;
  }
  sessionStorage.removeItem(TOKEN_KEY);
}

function Field({
  label,
  name,
  type,
  autoComplete,
}: {
  label: string;
  name: string;
  type: string;
  autoComplete: string;
}) {
  return (
    <label className="input w-full">
      <span className="label">{label}</span>
      <input name={name} type={type} autoComplete={autoComplete} required />
    </label>
  );
}

export function App() {
  const [apiBaseUrl, setApiBaseUrl] = useState<string | null>(null);
  const [email, setEmail] = useState<string | null>(null);
  const [notice, setNotice] = useState<Notice | null>(null);
  const [panel, setPanel] = useState<Panel>("login");
  const [ready, setReady] = useState(false);
  const timer = useRef<TimerHandle>({ id: null });
  const endSessionRef = useRef<() => void>(() => {});

  endSessionRef.current = () => {
    forgetToken(timer.current);
    setEmail(null);
  };

  useEffect(() => {
    const active = timer.current;
    let cancelled = false;

    async function loadIdentity(base: string, token: string) {
      const response = await fetch(`${base}/me`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (cancelled) {
        return;
      }
      if (response.status === 401) {
        endSessionRef.current();
        return;
      }
      if (!response.ok) {
        return;
      }
      const body = (await response.json()) as { email: string; token: string };
      if (cancelled) {
        return;
      }
      rememberToken(body.token, active, () => endSessionRef.current());
      setEmail(body.email);
    }

    void (async () => {
      try {
        const response = await fetch("/config.json");
        if (!response.ok) {
          throw new Error("could not load config");
        }
        const config = (await response.json()) as { apiBaseUrl?: string };
        if (!config.apiBaseUrl) {
          throw new Error("config is missing apiBaseUrl");
        }
        if (cancelled) {
          return;
        }
        const base = config.apiBaseUrl.replace(/\/$/, "");
        setApiBaseUrl(base);
        const existing = sessionStorage.getItem(TOKEN_KEY);
        if (existing) {
          await loadIdentity(base, existing);
        }
      } catch (error) {
        if (!cancelled) {
          setNotice({
            tone: "error",
            text: error instanceof Error ? error.message : "could not start",
          });
        }
      } finally {
        if (!cancelled) {
          setReady(true);
        }
      }
    })();

    return () => {
      cancelled = true;
      if (active.id != null) {
        window.clearTimeout(active.id);
        active.id = null;
      }
    };
  }, []);

  function showPanel(next: Panel) {
    setPanel(next);
    setNotice(null);
  }

  async function onRegister(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!apiBaseUrl) {
      return;
    }
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    try {
      const response = await fetch(`${apiBaseUrl}/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: String(form.get("email") ?? ""),
          password: String(form.get("password") ?? ""),
          confirmPassword: String(form.get("confirmPassword") ?? ""),
        }),
      });
      const body = (await response.json()) as { error?: string };
      if (!response.ok) {
        setNotice({ tone: "error", text: body.error ?? "could not register" });
        return;
      }
      formElement.reset();
      setPanel("login");
      setNotice({
        tone: "success",
        text: "Account created. Log in to continue.",
      });
    } catch (error) {
      setNotice({
        tone: "error",
        text: error instanceof Error ? error.message : "could not register",
      });
    }
  }

  async function onLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!apiBaseUrl) {
      return;
    }
    const form = new FormData(event.currentTarget);
    try {
      const response = await fetch(`${apiBaseUrl}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: String(form.get("email") ?? ""),
          password: String(form.get("password") ?? ""),
        }),
      });
      const body = (await response.json()) as {
        token?: string;
        error?: string;
      };
      if (!response.ok || !body.token) {
        setNotice({ tone: "error", text: body.error ?? "invalid credentials" });
        return;
      }
      rememberToken(body.token, timer.current, () => endSessionRef.current());
      const me = await fetch(`${apiBaseUrl}/me`, {
        headers: { Authorization: `Bearer ${body.token}` },
      });
      if (me.status === 401) {
        endSessionRef.current();
        setNotice({ tone: "error", text: "session rejected" });
        return;
      }
      if (!me.ok) {
        setNotice({ tone: "error", text: "could not load account" });
        return;
      }
      const identity = (await me.json()) as { email: string; token: string };
      rememberToken(identity.token, timer.current, () =>
        endSessionRef.current(),
      );
      setEmail(identity.email);
      setNotice(null);
    } catch (error) {
      setNotice({
        tone: "error",
        text: error instanceof Error ? error.message : "could not log in",
      });
    }
  }

  if (email) {
    return <Welcome email={email} />;
  }
  if (!ready) {
    return (
      <main className="flex min-h-dvh flex-col items-center justify-center bg-base-200 px-4 py-6 sm:px-6 md:px-8 lg:px-10">
        <p className="text-base-content">Loading…</p>
      </main>
    );
  }

  return (
    <main className="flex min-h-dvh flex-col items-center justify-center bg-base-200 px-4 py-6 sm:px-6 md:px-8 lg:px-10">
      <div className="grid w-full max-w-sm grid-cols-1 gap-4 sm:max-w-md md:max-w-lg">
        <section className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <div role="tablist" className="tabs tabs-box w-full">
              <input
                type="radio"
                name="auth-panel"
                className="tab grow"
                aria-label="Log in"
                checked={panel === "login"}
                onChange={() => showPanel("login")}
              />
              <input
                type="radio"
                name="auth-panel"
                className="tab grow"
                aria-label="Register"
                checked={panel === "register"}
                onChange={() => showPanel("register")}
              />
            </div>
            {panel === "login" ? (
              <form onSubmit={onLogin}>
                <fieldset className="fieldset">
                  <legend className="fieldset-legend">Log in</legend>
                  <Field
                    label="Email"
                    name="email"
                    type="email"
                    autoComplete="username"
                  />
                  <Field
                    label="Password"
                    name="password"
                    type="password"
                    autoComplete="current-password"
                  />
                  <div className="card-actions">
                    <button
                      className="btn w-full sm:w-auto"
                      type="submit"
                      disabled={!apiBaseUrl}
                    >
                      Log in
                    </button>
                  </div>
                </fieldset>
              </form>
            ) : (
              <form onSubmit={onRegister}>
                <fieldset className="fieldset">
                  <legend className="fieldset-legend">Register</legend>
                  <Field
                    label="Email"
                    name="email"
                    type="email"
                    autoComplete="email"
                  />
                  <Field
                    label="Password"
                    name="password"
                    type="password"
                    autoComplete="new-password"
                  />
                  <Field
                    label="Confirm password"
                    name="confirmPassword"
                    type="password"
                    autoComplete="new-password"
                  />
                  <div className="card-actions">
                    <button
                      className="btn w-full sm:w-auto"
                      type="submit"
                      disabled={!apiBaseUrl}
                    >
                      Create account
                    </button>
                  </div>
                </fieldset>
              </form>
            )}
          </div>
        </section>
        {notice ? (
          <div
            role="alert"
            className={`alert sm:alert-horizontal ${notice.tone === "error" ? "alert-error" : "alert-success"}`}
          >
            {notice.text}
          </div>
        ) : null}
      </div>
    </main>
  );
}
