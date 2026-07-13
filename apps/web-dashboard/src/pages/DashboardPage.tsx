import { useEffect, useState } from "react";
import { useAuth } from "../contexts/AuthContext";
import { enrollment, type Will, type LivenessStatus, type Trustee } from "../lib/api";
import {
  ScrollText,
  Users,
  HeartPulse,
  ShieldCheck,
  Activity,
  AlertTriangle,
  CheckCircle2,
  Clock,
} from "lucide-react";
import { Link } from "react-router-dom";

function lifecycleBadge(state: string) {
  const map: Record<string, { cls: string; label: string }> = {
    active: { cls: "badge-success", label: "Active" },
    pending_verification: { cls: "badge-warning", label: "Pending Verification" },
    grace_period: { cls: "badge-danger", label: "Grace Period" },
    ready_for_execution: { cls: "badge-info", label: "Ready for Execution" },
  };
  const m = map[state] ?? { cls: "badge-neutral", label: state };
  return <span className={`badge ${m.cls}`}>{m.label}</span>;
}

function timeAgo(dateStr: string | null): string {
  if (!dateStr) return "Never";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "Just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

export function DashboardPage() {
  const { user } = useAuth();
  const [will, setWill] = useState<Will | null>(null);
  const [liveness, setLiveness] = useState<LivenessStatus | null>(null);
  const [trustees, setTrustees] = useState<Trustee[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.allSettled([
      enrollment.getWill(),
      enrollment.getLiveness(),
      enrollment.listTrustees(),
    ]).then(([w, l, t]) => {
      if (w.status === "fulfilled") setWill(w.value);
      if (l.status === "fulfilled") setLiveness(l.value);
      if (t.status === "fulfilled") setTrustees(t.value.trustees ?? []);
      setLoading(false);
    });
  }, []);

  if (loading)
    return (
      <div className="page-container loading-screen" style={{ minHeight: "60vh" }}>
        <div className="spinner spinner-lg" />
      </div>
    );

  return (
    <div className="page-container animate-in">
      <div className="page-header">
        <h1>
          Welcome{user?.displayName ? `, ${user.displayName}` : ""}
        </h1>
        <p>Your digital legacy overview</p>
      </div>

      {/* Lifecycle State Banner */}
      {liveness && liveness.lifecycleState !== "active" && (
        <div
          className="glass-card"
          style={{
            padding: 20,
            marginBottom: 24,
            display: "flex",
            alignItems: "center",
            gap: 16,
            borderColor: liveness.lifecycleState === "grace_period"
              ? "rgba(239,68,68,0.3)"
              : "rgba(245,158,11,0.3)",
          }}
        >
          <AlertTriangle
            size={24}
            style={{
              color: liveness.lifecycleState === "grace_period"
                ? "var(--color-danger)"
                : "var(--color-warning)",
            }}
          />
          <div>
            <strong style={{ fontSize: "0.9rem" }}>
              Your will is in {liveness.lifecycleState.replace(/_/g, " ")} state
            </strong>
            <p style={{ fontSize: "0.8rem", color: "var(--color-text-secondary)", marginTop: 2 }}>
              Send a heartbeat to prove liveness and return to active state.
            </p>
          </div>
          <Link to="/heartbeat" className="btn btn-primary btn-sm" style={{ marginLeft: "auto" }}>
            Send Heartbeat
          </Link>
        </div>
      )}

      {/* Stat Cards */}
      <div className="grid-3" style={{ marginBottom: 28 }}>
        <div className="glass-card stat-card">
          <div className="stat-icon" style={{ background: "var(--color-info-dim)" }}>
            <ScrollText size={20} style={{ color: "var(--color-info)" }} />
          </div>
          <div className="stat-value">{will ? `v${will.version}` : "—"}</div>
          <div className="stat-label">Will Version</div>
          <div style={{ marginTop: 8 }}>
            {will ? (
              <span className={`badge ${will.state === "published" ? "badge-success" : "badge-neutral"}`}>
                {will.state}
              </span>
            ) : (
              <span className="badge badge-neutral">No will</span>
            )}
          </div>
        </div>

        <div className="glass-card stat-card">
          <div className="stat-icon" style={{ background: "var(--color-success-dim)" }}>
            <Users size={20} style={{ color: "var(--color-success)" }} />
          </div>
          <div className="stat-value">{trustees.length}</div>
          <div className="stat-label">Trustees</div>
        </div>

        <div className="glass-card stat-card">
          <div className="stat-icon" style={{ background: "var(--color-warning-dim)" }}>
            <HeartPulse size={20} style={{ color: "var(--color-warning)" }} />
          </div>
          <div className="stat-value">{timeAgo(liveness?.lastHeartbeatAt ?? null)}</div>
          <div className="stat-label">Last Heartbeat</div>
        </div>
      </div>

      {/* Lifecycle + Quick Actions */}
      <div className="grid-2">
        <div className="glass-card" style={{ padding: 24 }}>
          <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 16, display: "flex", alignItems: "center", gap: 8 }}>
            <Activity size={18} style={{ color: "var(--color-accent)" }} />
            Lifecycle Status
          </h3>
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <span style={{ fontSize: "0.85rem", color: "var(--color-text-secondary)" }}>Current State</span>
              {liveness ? lifecycleBadge(liveness.lifecycleState) : <span className="badge badge-neutral">Unknown</span>}
            </div>
            {will && (
              <>
                <div style={{ display: "flex", justifyContent: "space-between" }}>
                  <span style={{ fontSize: "0.85rem", color: "var(--color-text-secondary)" }}>Dormancy</span>
                  <span style={{ fontSize: "0.85rem" }}>{will.dormancyPeriodDays} days</span>
                </div>
                <div style={{ display: "flex", justifyContent: "space-between" }}>
                  <span style={{ fontSize: "0.85rem", color: "var(--color-text-secondary)" }}>Grace Period</span>
                  <span style={{ fontSize: "0.85rem" }}>{will.gracePeriodDays} days</span>
                </div>
                <div style={{ display: "flex", justifyContent: "space-between" }}>
                  <span style={{ fontSize: "0.85rem", color: "var(--color-text-secondary)" }}>Categories</span>
                  <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                    {will.releaseCategories?.map((c) => (
                      <span key={c} className="badge badge-neutral" style={{ fontSize: "0.65rem" }}>
                        {c}
                      </span>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>

        <div className="glass-card" style={{ padding: 24 }}>
          <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 16, display: "flex", alignItems: "center", gap: 8 }}>
            <ShieldCheck size={18} style={{ color: "var(--color-accent)" }} />
            Quick Actions
          </h3>
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            <Link to="/heartbeat" className="btn btn-secondary" style={{ justifyContent: "flex-start" }}>
              <HeartPulse size={16} /> Send Heartbeat
            </Link>
            {!will && (
              <Link to="/will" className="btn btn-primary" style={{ justifyContent: "flex-start" }}>
                <ScrollText size={16} /> Create Your Will
              </Link>
            )}
            <Link to="/trustees" className="btn btn-secondary" style={{ justifyContent: "flex-start" }}>
              <Users size={16} /> Manage Trustees
            </Link>
            <Link to="/audit" className="btn btn-secondary" style={{ justifyContent: "flex-start" }}>
              <CheckCircle2 size={16} /> View Audit Log
            </Link>
            <Link to="/verifications" className="btn btn-secondary" style={{ justifyContent: "flex-start" }}>
              <Clock size={16} /> Pending Verifications
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
