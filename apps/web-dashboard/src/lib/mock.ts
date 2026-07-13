/**
 * Mock API layer — used when backend services are not running.
 * Stores everything in localStorage so the demo persists across refreshes.
 */

import { v4 } from "./uuid";

// ── helpers ──
function store<T>(key: string, val: T) {
  localStorage.setItem(`waaris_mock_${key}`, JSON.stringify(val));
}
function load<T>(key: string, fallback: T): T {
  const raw = localStorage.getItem(`waaris_mock_${key}`);
  return raw ? JSON.parse(raw) : fallback;
}

const delay = (ms = 300) => new Promise((r) => setTimeout(r, ms));

// ── mock user state ──
export interface MockUser {
  id: string;
  email: string;
  displayName: string;
  password: string;
  createdAt: string;
}

function getUsers(): MockUser[] {
  return load<MockUser[]>("users", []);
}
function saveUsers(u: MockUser[]) {
  store("users", u);
}
function currentUserId(): string | null {
  return localStorage.getItem("waaris_mock_current_user");
}
function setCurrentUser(id: string) {
  localStorage.setItem("waaris_mock_current_user", id);
}
function clearCurrentUser() {
  localStorage.removeItem("waaris_mock_current_user");
}

// ── mock data factories ──
function makeSession(u: MockUser) {
  return {
    user: { id: u.id, email: u.email, displayName: u.displayName, createdAt: u.createdAt },
    accessToken: `mock_at_${u.id}_${Date.now()}`,
    refreshToken: `mock_rt_${u.id}_${Date.now()}`,
    accessTokenExpiresAt: new Date(Date.now() + 900_000).toISOString(),
  };
}

// ── Auth Mock ──
export const mockAuth = {
  async register(email: string, password: string, displayName?: string) {
    await delay();
    const users = getUsers();
    if (users.find((u) => u.email === email.toLowerCase())) {
      throw { status: 409, code: "conflict", message: "An account with this email already exists" };
    }
    const user: MockUser = {
      id: v4(),
      email: email.toLowerCase(),
      displayName: displayName || email.split("@")[0],
      password,
      createdAt: new Date().toISOString(),
    };
    users.push(user);
    saveUsers(users);
    setCurrentUser(user.id);
    return makeSession(user);
  },

  async login(email: string, password: string) {
    await delay();
    const user = getUsers().find((u) => u.email === email.toLowerCase() && u.password === password);
    if (!user) throw { status: 401, code: "unauthorized", message: "Invalid email or password" };
    setCurrentUser(user.id);
    return makeSession(user);
  },

  async refresh(_rt: string) {
    await delay(100);
    const uid = currentUserId();
    const user = uid ? getUsers().find((u) => u.id === uid) : null;
    if (!user) throw { status: 401, code: "unauthorized", message: "Session expired" };
    return makeSession(user);
  },

  async logout(_rt: string) {
    await delay(50);
    clearCurrentUser();
  },

  async me() {
    await delay(100);
    const uid = currentUserId();
    const user = uid ? getUsers().find((u) => u.id === uid) : null;
    if (!user) throw { status: 401, code: "unauthorized", message: "Not authenticated" };
    return { id: user.id, email: user.email, displayName: user.displayName, createdAt: user.createdAt };
  },

  async updateMe(displayName: string) {
    await delay();
    const uid = currentUserId();
    const users = getUsers();
    const user = uid ? users.find((u) => u.id === uid) : null;
    if (!user) throw { status: 401, code: "unauthorized", message: "Not authenticated" };
    user.displayName = displayName;
    saveUsers(users);
    return { id: user.id, email: user.email, displayName: user.displayName, createdAt: user.createdAt };
  },

  async deleteMe() {
    await delay();
    const uid = currentUserId();
    const users = getUsers().filter((u) => u.id !== uid);
    saveUsers(users);
    clearCurrentUser();
    store("wills", load<Record<string, unknown>[]>("wills", []).filter((w: Record<string, unknown>) => w.userId !== uid));
  },
};

// ── Enrollment Mock ──
interface MockWill {
  id: string;
  userId: string;
  state: "draft" | "published";
  version: number;
  dormancyPeriodDays: number;
  gracePeriodDays: number;
  policyVersionAccepted: string;
  consentAcceptedAt: string;
  releaseCategories: string[];
  lifecycleState: string;
  createdAt: string;
  updatedAt: string;
  deleted: boolean;
}

interface MockTrustee {
  id: string;
  willId: string;
  userId: string;
  name: string;
  email: string;
  relationship: string;
  createdAt: string;
  updatedAt: string;
}

interface MockHeartbeat {
  id: string;
  willId: string;
  userId: string;
  source: string;
  occurredAt: string;
  createdAt: string;
}

function getWill(): MockWill | null {
  const uid = currentUserId();
  return load<MockWill[]>("wills", []).find((w) => w.userId === uid && !w.deleted) ?? null;
}
function saveWill(w: MockWill) {
  const wills = load<MockWill[]>("wills", []).filter(
    (x) => !(x.userId === w.userId && !x.deleted),
  );
  wills.push(w);
  store("wills", wills);
}
function getTrustees(): MockTrustee[] {
  const w = getWill();
  return w ? load<MockTrustee[]>("trustees", []).filter((t) => t.willId === w.id) : [];
}
function saveTrustees(ts: MockTrustee[]) {
  const w = getWill();
  const others = load<MockTrustee[]>("trustees", []).filter((t) => t.willId !== w?.id);
  store("trustees", [...others, ...ts]);
}
function getHeartbeats(): MockHeartbeat[] {
  const w = getWill();
  return w ? load<MockHeartbeat[]>("heartbeats", []).filter((h) => h.willId === w.id) : [];
}

export const mockEnrollment = {
  async createWill(data: Partial<MockWill>) {
    await delay();
    const uid = currentUserId()!;
    if (getWill()) throw { status: 409, code: "conflict", message: "Active will already exists" };
    const now = new Date().toISOString();
    const w: MockWill = {
      id: v4(),
      userId: uid,
      state: (data.state as "draft" | "published") || "draft",
      version: 1,
      dormancyPeriodDays: data.dormancyPeriodDays ?? 180,
      gracePeriodDays: data.gracePeriodDays ?? 30,
      policyVersionAccepted: data.policyVersionAccepted ?? "2026-07",
      consentAcceptedAt: now,
      releaseCategories: data.releaseCategories ?? ["financial", "private"],
      lifecycleState: "active",
      createdAt: now,
      updatedAt: now,
      deleted: false,
    };
    saveWill(w);
    store("will_versions", [...load<MockWill[]>("will_versions", []), { ...w }]);
    return w;
  },

  async getWill() {
    await delay(100);
    const w = getWill();
    if (!w) throw { status: 404, code: "not_found", message: "No active will" };
    return w;
  },

  async updateWill(data: Partial<MockWill>) {
    await delay();
    const w = getWill();
    if (!w) throw { status: 404, code: "not_found", message: "No active will" };
    const now = new Date().toISOString();
    w.state = (data.state as "draft" | "published") || w.state;
    w.dormancyPeriodDays = data.dormancyPeriodDays ?? w.dormancyPeriodDays;
    w.gracePeriodDays = data.gracePeriodDays ?? w.gracePeriodDays;
    w.policyVersionAccepted = data.policyVersionAccepted ?? w.policyVersionAccepted;
    w.releaseCategories = data.releaseCategories ?? w.releaseCategories;
    w.version += 1;
    w.updatedAt = now;
    w.consentAcceptedAt = now;
    saveWill(w);
    store("will_versions", [...load<MockWill[]>("will_versions", []), { ...w }]);
    return w;
  },

  async deleteWill() {
    await delay();
    const w = getWill();
    if (!w) throw { status: 404, code: "not_found", message: "No active will" };
    w.deleted = true;
    const wills = load<MockWill[]>("wills", []).map((x) => (x.id === w.id ? w : x));
    store("wills", wills);
  },

  async getHistory() {
    await delay(100);
    const w = getWill();
    const versions = load<MockWill[]>("will_versions", []).filter((v) => v.userId === currentUserId() && v.id === w?.id);
    return { history: versions };
  },

  async addTrustee(data: { name: string; email: string; relationship: string }) {
    await delay();
    const w = getWill();
    if (!w) throw { status: 404, code: "not_found", message: "Create a will first" };
    const uid = currentUserId()!;
    const users = getUsers();
    const ownerEmail = users.find((u) => u.id === uid)?.email;
    if (data.email.toLowerCase() === ownerEmail)
      throw { status: 400, code: "bad_request", message: "Cannot add yourself as a trustee" };
    const existing = getTrustees();
    if (existing.find((t) => t.email === data.email.toLowerCase()))
      throw { status: 409, code: "conflict", message: "Trustee with this email already exists" };
    const now = new Date().toISOString();
    const t: MockTrustee = {
      id: v4(),
      willId: w.id,
      userId: uid,
      name: data.name,
      email: data.email.toLowerCase(),
      relationship: data.relationship,
      createdAt: now,
      updatedAt: now,
    };
    saveTrustees([...existing, t]);
    return t;
  },

  async listTrustees() {
    await delay(100);
    return { trustees: getTrustees() };
  },

  async updateTrustee(id: string, data: { name: string; email: string; relationship: string }) {
    await delay();
    const ts = getTrustees();
    const t = ts.find((x) => x.id === id);
    if (!t) throw { status: 404, code: "not_found", message: "Trustee not found" };
    t.name = data.name;
    t.email = data.email.toLowerCase();
    t.relationship = data.relationship;
    t.updatedAt = new Date().toISOString();
    saveTrustees(ts);
    return t;
  },

  async deleteTrustee(id: string) {
    await delay();
    const ts = getTrustees().filter((t) => t.id !== id);
    saveTrustees(ts);
  },

  async sendHeartbeat() {
    await delay();
    const w = getWill();
    if (!w) throw { status: 404, code: "not_found", message: "No active will" };
    const now = new Date().toISOString();
    const hb: MockHeartbeat = { id: v4(), willId: w.id, userId: currentUserId()!, source: "web", occurredAt: now, createdAt: now };
    store("heartbeats", [...load<MockHeartbeat[]>("heartbeats", []), hb]);
    w.lifecycleState = "active";
    saveWill(w);
    return {
      willId: w.id,
      lifecycleState: "active",
      lastHeartbeatAt: now,
      pendingVerificationStartedAt: null,
      gracePeriodStartedAt: null,
      readyForExecutionAt: null,
    };
  },

  async getLiveness() {
    await delay(100);
    const w = getWill();
    if (!w) throw { status: 404, code: "not_found", message: "No active will" };
    const hbs = getHeartbeats();
    const last = hbs.length ? hbs[hbs.length - 1].occurredAt : null;
    return {
      willId: w.id,
      lifecycleState: w.lifecycleState,
      lastHeartbeatAt: last,
      pendingVerificationStartedAt: null,
      gracePeriodStartedAt: null,
      readyForExecutionAt: null,
    };
  },

  async getHeartbeatHistory() {
    await delay(100);
    return { history: getHeartbeats().reverse() };
  },

  async getPendingVerifications() {
    await delay(100);
    return { pending: [] };
  },

  async approve(_id: string) { await delay(); },
  async reject(_id: string) { await delay(); },
  async abstain(_id: string) { await delay(); },

  async getNotifications() {
    await delay(100);
    return { history: load("notifications", []) };
  },

  async getAuditHistory() {
    await delay(100);
    const uid = currentUserId();
    const events = [
      { id: v4(), userId: uid, willId: null, actor: "system", event: "user.registered", correlationId: v4(), details: "Account created via web dashboard (demo mode)", createdAt: new Date(Date.now() - 86400000).toISOString() },
      { id: v4(), userId: uid, willId: null, actor: "user", event: "user.login", correlationId: v4(), details: "Login from web dashboard", createdAt: new Date().toISOString() },
    ];
    return { history: events };
  },
};
