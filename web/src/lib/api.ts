export const TOKEN_KEY = "msf_token";
export const REFRESH_TOKEN_KEY = "msf_refresh_token";

export interface ApiErrorPayload {
  error?: string;
  message?: string;
  success?: boolean;
}

export class ApiError extends Error {
  status: number;
  code: string;
  payload: unknown;

  constructor(status: number, code: string, message: string, payload: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.payload = payload;
  }
}

type ApiOptions = RequestInit & {
  skipAuth?: boolean;
};

export function getToken() {
  return window.localStorage.getItem(TOKEN_KEY) || "";
}

export function getRefreshToken() {
  return window.localStorage.getItem(REFRESH_TOKEN_KEY) || "";
}

export function setSession(token: string, refreshToken?: string) {
  window.localStorage.setItem(TOKEN_KEY, token);
  if (refreshToken) {
    window.localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  }
}

export function clearSession() {
  window.localStorage.removeItem(TOKEN_KEY);
  window.localStorage.removeItem(REFRESH_TOKEN_KEY);
  window.localStorage.removeItem("msf-auth");
}

function parsePayloadMessage(payload: unknown, fallback: string) {
  if (payload && typeof payload === "object") {
    const data = payload as ApiErrorPayload;
    return data.message || data.error || fallback;
  }
  if (typeof payload === "string" && payload.trim()) {
    return payload;
  }
  return fallback;
}

export async function api<T = any>(path: string, options: ApiOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body && !(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const token = getToken();
  if (token && !options.skipAuth && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(path, { ...options, headers });
  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json")
    ? await response.json().catch(() => null)
    : await response.text().catch(() => "");

  if (!response.ok) {
    if (response.status === 401) {
      clearSession();
      window.dispatchEvent(new Event("msf-auth-expired"));
    }
    const code =
      payload && typeof payload === "object" && "error" in payload
        ? String((payload as ApiErrorPayload).error)
        : response.statusText;
    throw new ApiError(response.status, code, parsePayloadMessage(payload, response.statusText), payload);
  }

  return payload as T;
}

export function apiData<T = any>(payload: any, fallback?: T): T {
  if (payload && typeof payload === "object" && "data" in payload) {
    return payload.data as T;
  }
  return (fallback ?? payload) as T;
}

export function apiList<T = any>(payload: any, keys: string[] = ["data", "items", "logs", "services"]): T[] {
  if (Array.isArray(payload)) return payload as T[];
  if (!payload || typeof payload !== "object") return [];
  for (const key of keys) {
    if (Array.isArray(payload[key])) return payload[key] as T[];
  }
  return [];
}

export function formatBytes(value: unknown) {
  const bytes = Number(value || 0);
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = bytes;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(size >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

export function formatPercent(value: unknown) {
  const numeric = Number(value || 0);
  if (!Number.isFinite(numeric)) return "0.0%";
  return `${numeric.toFixed(1)}%`;
}
