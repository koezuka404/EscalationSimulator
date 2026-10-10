import type { LoginResult } from "./types";

const csrfCookieName = "escalator_csrf_token";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

let accessToken = "";
let refreshInFlight: Promise<LoginResult | null> | null = null;

export function setAccessToken(token: string) {
  accessToken = token;
}

export function getAccessToken(): string {
  return accessToken;
}

export function readCookie(name: string): string {
  for (const part of document.cookie.split("; ")) {
    const index = part.indexOf("=");
    if (index === -1) continue;
    if (part.slice(0, index) === name) return decodeURIComponent(part.slice(index + 1));
  }
  return "";
}

async function readBody(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return null;
  }
}

function messageFrom(body: unknown, fallback: string): string {
  if (body && typeof body === "object" && "message" in body && typeof body.message === "string" && body.message) {
    return body.message;
  }
  return fallback;
}

async function send(path: string, init: RequestInit, token: string): Promise<Response> {
  const headers = new Headers(init.headers);
  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  return fetch(path, { ...init, headers, credentials: "include" });
}

export async function refreshSession(): Promise<LoginResult | null> {
  if (!refreshInFlight) {
    refreshInFlight = refreshOnce().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function refreshOnce(): Promise<LoginResult | null> {
  const csrf = readCookie(csrfCookieName);
  if (!csrf) return null;
  let response: Response;
  try {
    response = await fetch("/api/auth/refresh", {
      method: "POST",
      credentials: "include",
      headers: { "X-CSRF-Token": csrf },
    });
  } catch {
    return null;
  }
  if (!response.ok) return null;
  const body = (await readBody(response)) as LoginResult | null;
  if (!body?.access_token || !body.user) return null;
  accessToken = body.access_token;
  return body;
}

export async function api<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  let response: Response;
  try {
    response = await send(path, init, accessToken);
  } catch {
    throw new ApiError(0, "通信できませんでした。しばらくしてから、もう一度試してください");
  }
  if (response.status === 401 && retry && path !== "/api/auth/login" && path !== "/api/auth/refresh") {
    const renewed = await refreshSession();
    if (renewed) return api<T>(path, init, false);
  }
  const body = await readBody(response);
  if (!response.ok) {
    throw new ApiError(response.status, messageFrom(body, "操作を完了できませんでした。しばらくしてから、もう一度試してください"));
  }
  return body as T;
}

export async function login(email: string, password: string): Promise<LoginResult> {
  let response: Response;
  try {
    response = await fetch("/api/auth/login", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
  } catch {
    throw new ApiError(0, "通信できませんでした。しばらくしてから、もう一度試してください");
  }
  const body = await readBody(response);
  if (!response.ok) {
    throw new ApiError(response.status, messageFrom(body, "ログインできませんでした。しばらくしてから、もう一度試してください"));
  }
  const result = body as LoginResult;
  accessToken = result.access_token;
  return result;
}

export async function logout(): Promise<void> {
  const csrf = readCookie(csrfCookieName);
  try {
    await fetch("/api/auth/logout", {
      method: "POST",
      credentials: "include",
      headers: csrf ? { "X-CSRF-Token": csrf } : {},
    });
  } catch {
    // 画面側のログイン状態は消す
  }
  accessToken = "";
}
