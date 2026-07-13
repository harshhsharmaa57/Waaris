import { mockAuth, mockEnrollment } from "./mock";

const AUTH_BASE = import.meta.env.VITE_AUTH_BASE_URL ?? "http://localhost:8080";
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8081";

/**
 * Detect if the backend is reachable. Cached after first probe.
 * When the backend is down (Docker not running), automatically use mock mode.
 */
let _backendAvailable: boolean | null = null;

async function isBackendAvailable(): Promise<boolean> {
  if (_backendAvailable !== null) return _backendAvailable;
  try {
    const res = await fetch(`${AUTH_BASE}/healthz`, { signal: AbortSignal.timeout(2000) });
    _backendAvailable = res.ok;
  } catch {
    _backendAvailable = false;
  }
  if (!_backendAvailable) {
    console.info(
      "%c⚡ Waaris Demo Mode %c— Backend not reachable, using local mock API",
      "color:#f59e0b;font-weight:bold",
      "color:#8b95b0",
    );
  }
  return _backendAvailable;
}

// ── shared helpers ──
interface ApiError {
  code: string;
  message: string;
  correlationId?: string;
}

export class ApiClientError extends Error {
  status: number;
  code: string;
  correlationId?: string;
  constructor(status: number, body: ApiError) {
    super(body.message);
    this.status = status;
    this.code = body.code;
    this.correlationId = body.correlationId;
  }
}

function getToken(): string | null {
  return localStorage.getItem("waaris_access_token");
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem("waaris_access_token", access);
  localStorage.setItem("waaris_refresh_token", refresh);
}

export function clearTokens() {
  localStorage.removeItem("waaris_access_token");
  localStorage.removeItem("waaris_refresh_token");
}

export function getRefreshToken(): string | null {
  return localStorage.getItem("waaris_refresh_token");
}

async function request<T>(
  base: string,
  path: string,
  opts: RequestInit = {},
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(opts.headers as Record<string, string>),
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${base}${path}`, { ...opts, headers });

  if (res.status === 204) return undefined as T;

  const body = await res.json().catch(() => null);

  if (!res.ok) {
    throw new ApiClientError(res.status, body ?? { code: "unknown", message: res.statusText });
  }
  return body as T;
}

/** Wraps a mock call to throw ApiClientError like the real API does */
async function mockCall<T>(fn: () => Promise<T>): Promise<T> {
  try {
    return await fn();
  } catch (err: unknown) {
    if (err && typeof err === "object" && "status" in err) {
      const e = err as { status: number; code: string; message: string };
      throw new ApiClientError(e.status, { code: e.code, message: e.message });
    }
    throw err;
  }
}

// ── Auth ──
export interface SessionResponse {
  user: { id: string; email: string; displayName: string; createdAt: string };
  accessToken: string;
  refreshToken: string;
  accessTokenExpiresAt: string;
}

export const auth = {
  register: async (email: string, password: string, displayName?: string) => {
    if (!(await isBackendAvailable()))
      return mockCall(() => mockAuth.register(email, password, displayName)) as Promise<SessionResponse>;
    return request<SessionResponse>(AUTH_BASE, "/v1/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, displayName }),
    });
  },
  login: async (email: string, password: string) => {
    if (!(await isBackendAvailable()))
      return mockCall(() => mockAuth.login(email, password)) as Promise<SessionResponse>;
    return request<SessionResponse>(AUTH_BASE, "/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  },
  refresh: async (refreshToken: string) => {
    if (!(await isBackendAvailable()))
      return mockCall(() => mockAuth.refresh(refreshToken)) as Promise<SessionResponse>;
    return request<SessionResponse>(AUTH_BASE, "/v1/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refreshToken }),
    });
  },
  logout: async (refreshToken: string) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockAuth.logout(refreshToken)) as Promise<void>;
    return request<void>(AUTH_BASE, "/v1/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refreshToken }),
    });
  },
  me: async () => {
    if (!(await isBackendAvailable()))
      return mockCall(() => mockAuth.me()) as Promise<SessionResponse["user"]>;
    return request<SessionResponse["user"]>(AUTH_BASE, "/v1/users/me");
  },
  updateMe: async (displayName: string) => {
    if (!(await isBackendAvailable()))
      return mockCall(() => mockAuth.updateMe(displayName)) as Promise<SessionResponse["user"]>;
    return request<SessionResponse["user"]>(AUTH_BASE, "/v1/users/me", {
      method: "PATCH",
      body: JSON.stringify({ displayName }),
    });
  },
  deleteMe: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockAuth.deleteMe()) as Promise<void>;
    return request<void>(AUTH_BASE, "/v1/users/me", { method: "DELETE" });
  },
};

// ── Enrollment types ──
export interface Will {
  id: string;
  userId: string;
  state: "draft" | "published";
  version: number;
  dormancyPeriodDays: number;
  gracePeriodDays: number;
  policyVersionAccepted: string;
  consentAcceptedAt: string;
  releaseCategories: string[];
  createdAt: string;
  updatedAt: string;
}

export interface Trustee {
  id: string;
  willId: string;
  userId: string;
  name: string;
  email: string;
  relationship: string;
  createdAt: string;
  updatedAt: string;
}

export interface LivenessStatus {
  willId: string;
  lifecycleState: string;
  lastHeartbeatAt: string | null;
  pendingVerificationStartedAt: string | null;
  gracePeriodStartedAt: string | null;
  readyForExecutionAt: string | null;
}

export interface AuditEvent {
  id: string;
  userId: string | null;
  willId: string | null;
  actor: string;
  event: string;
  correlationId: string;
  details: string;
  createdAt: string;
}

export interface Notification {
  id: string;
  willId: string;
  status: string;
  createdAt: string;
}

export interface VerificationPending {
  id: string;
  willId: string;
  status: string;
}

// ── Enrollment ──
export const enrollment = {
  createWill: async (data: Partial<Will>) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.createWill(data)) as Promise<Will>;
    return request<Will>(API_BASE, "/api/v1/will", { method: "POST", body: JSON.stringify(data) });
  },
  getWill: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.getWill()) as Promise<Will>;
    return request<Will>(API_BASE, "/api/v1/will");
  },
  updateWill: async (data: Partial<Will>) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.updateWill(data)) as Promise<Will>;
    return request<Will>(API_BASE, "/api/v1/will", { method: "PUT", body: JSON.stringify(data) });
  },
  deleteWill: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.deleteWill()) as Promise<void>;
    return request<void>(API_BASE, "/api/v1/will", { method: "DELETE" });
  },
  getHistory: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.getHistory()) as Promise<{ history: Will[] }>;
    return request<{ history: Will[] }>(API_BASE, "/api/v1/will/history");
  },

  addTrustee: async (data: { name: string; email: string; relationship: string }) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.addTrustee(data)) as Promise<Trustee>;
    return request<Trustee>(API_BASE, "/api/v1/trustees", { method: "POST", body: JSON.stringify(data) });
  },
  listTrustees: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.listTrustees()) as Promise<{ trustees: Trustee[] }>;
    return request<{ trustees: Trustee[] }>(API_BASE, "/api/v1/trustees");
  },
  updateTrustee: async (id: string, data: { name: string; email: string; relationship: string }) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.updateTrustee(id, data)) as Promise<Trustee>;
    return request<Trustee>(API_BASE, `/api/v1/trustees/${id}`, { method: "PUT", body: JSON.stringify(data) });
  },
  deleteTrustee: async (id: string) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.deleteTrustee(id)) as Promise<void>;
    return request<void>(API_BASE, `/api/v1/trustees/${id}`, { method: "DELETE" });
  },

  sendHeartbeat: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.sendHeartbeat()) as Promise<LivenessStatus>;
    return request<LivenessStatus>(API_BASE, "/api/v1/heartbeats", { method: "POST" });
  },
  getLiveness: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.getLiveness()) as Promise<LivenessStatus>;
    return request<LivenessStatus>(API_BASE, "/api/v1/heartbeats");
  },
  getHeartbeatHistory: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.getHeartbeatHistory()) as Promise<{ history: Array<{ id: string; occurredAt: string; source: string }> }>;
    return request<{ history: Array<{ id: string; occurredAt: string; source: string }> }>(API_BASE, "/api/v1/heartbeats/history");
  },

  getPendingVerifications: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.getPendingVerifications()) as Promise<{ pending: VerificationPending[] }>;
    return request<{ pending: VerificationPending[] }>(API_BASE, "/api/v1/verifications/pending");
  },
  approve: async (id: string) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.approve(id)) as Promise<void>;
    return request<void>(API_BASE, `/api/v1/verifications/${id}/approve`, { method: "POST" });
  },
  reject: async (id: string) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.reject(id)) as Promise<void>;
    return request<void>(API_BASE, `/api/v1/verifications/${id}/reject`, { method: "POST" });
  },
  abstain: async (id: string) => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.abstain(id)) as Promise<void>;
    return request<void>(API_BASE, `/api/v1/verifications/${id}/abstain`, { method: "POST" });
  },

  getNotifications: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.getNotifications()) as Promise<{ history: Notification[] }>;
    return request<{ history: Notification[] }>(API_BASE, "/api/v1/notifications/history");
  },
  getAuditHistory: async () => {
    if (!(await isBackendAvailable())) return mockCall(() => mockEnrollment.getAuditHistory()) as Promise<{ history: AuditEvent[] }>;
    return request<{ history: AuditEvent[] }>(API_BASE, "/api/v1/audit/history");
  },
};
