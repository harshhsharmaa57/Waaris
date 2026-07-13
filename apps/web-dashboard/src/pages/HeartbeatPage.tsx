import { useState, useEffect } from "react";
import { enrollment, type LivenessStatus, ApiClientError } from "../lib/api";
import { HeartPulse, Activity, Zap, Clock, CheckCircle2 } from "lucide-react";
import toast from "react-hot-toast";

export function HeartbeatPage() {
  const [liveness, setLiveness] = useState<LivenessStatus | null>(null);
  const [history, setHistory] = useState<Array<{ id: string; occurredAt: string; source: string }>>([]);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [pulseAnim, setPulseAnim] = useState(false);

  const load = async () => {
    try {
      const [l, h] = await Promise.all([
        enrollment.getLiveness(),
        enrollment.getHeartbeatHistory(),
      ]);
      setLiveness(l);
      setHistory(h.history ?? []);
    } catch { /* empty */ }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const handleSend = async () => {
    setSending(true);
    try {
      const l = await enrollment.sendHeartbeat();
      setLiveness(l);
      setPulseAnim(true);
      setTimeout(() => setPulseAnim(false), 1500);
      toast.success("Heartbeat sent successfully");
      const h = await enrollment.getHeartbeatHistory();
      setHistory(h.history ?? []);
    } catch (err) {
      toast.error(err instanceof ApiClientError ? err.message : "Failed to send heartbeat");
    } finally {
      setSending(false);
    }
  };

  if (loading)
    return (
      <div className="page-container" style={{ display: "flex", justifyContent: "center", paddingTop: 80 }}>
        <div className="spinner spinner-lg" />
      </div>
    );

  const stateColor =
    liveness?.lifecycleState === "active"
      ? "var(--color-success)"
      : liveness?.lifecycleState === "grace_period"
        ? "var(--color-danger)"
        : "var(--color-warning)";

  return (
    <div className="page-container animate-in">
      <div className="page-header">
        <h1>Heartbeat</h1>
        <p>Prove liveness to keep your will in active state</p>
      </div>

      {/* Pulse Section */}
      <div className="glass-card" style={{ padding: 40, textAlign: "center", marginBottom: 28, position: "relative", overflow: "hidden" }}>
        {/* Pulse rings */}
        {pulseAnim && (
          <>
            <div style={{ ...pulseRingStyle, animationDelay: "0s" }} />
            <div style={{ ...pulseRingStyle, animationDelay: "0.4s" }} />
            <div style={{ ...pulseRingStyle, animationDelay: "0.8s" }} />
          </>
        )}

        <div style={{
          width: 80, height: 80, borderRadius: "50%",
          background: `${stateColor}20`,
          display: "flex", alignItems: "center", justifyContent: "center",
          margin: "0 auto 20px",
          border: `2px solid ${stateColor}40`,
          position: "relative", zIndex: 1,
        }}>
          <HeartPulse size={36} style={{ color: stateColor }} />
        </div>
        <div style={{ fontSize: "0.8rem", color: "var(--color-text-secondary)", marginBottom: 4 }}>Current State</div>
        <div style={{ fontSize: "1.2rem", fontWeight: 700, textTransform: "capitalize", color: stateColor, marginBottom: 20 }}>
          {liveness?.lifecycleState?.replace(/_/g, " ") ?? "Unknown"}
        </div>
        <button
          className="btn btn-primary btn-lg"
          onClick={handleSend}
          disabled={sending}
          style={{ minWidth: 200, position: "relative", zIndex: 1 }}
        >
          {sending ? <div className="spinner" style={{ borderTopColor: "#0a0e1a" }} /> : <><Zap size={18} /> Send Heartbeat</>}
        </button>
      </div>

      <div className="grid-2">
        {/* Liveness Details */}
        <div className="glass-card" style={{ padding: 24 }}>
          <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 16, display: "flex", alignItems: "center", gap: 8 }}>
            <Activity size={18} style={{ color: "var(--color-accent)" }} />
            Liveness Details
          </h3>
          <div style={{ display: "flex", flexDirection: "column", gap: 12, fontSize: "0.85rem" }}>
            <DetailRow label="Last Heartbeat" value={liveness?.lastHeartbeatAt ? new Date(liveness.lastHeartbeatAt).toLocaleString() : "Never"} />
            <DetailRow label="Pending Since" value={liveness?.pendingVerificationStartedAt ? new Date(liveness.pendingVerificationStartedAt).toLocaleString() : "—"} />
            <DetailRow label="Grace Started" value={liveness?.gracePeriodStartedAt ? new Date(liveness.gracePeriodStartedAt).toLocaleString() : "—"} />
            <DetailRow label="Ready At" value={liveness?.readyForExecutionAt ? new Date(liveness.readyForExecutionAt).toLocaleString() : "—"} />
          </div>
        </div>

        {/* History */}
        <div className="glass-card" style={{ padding: 24 }}>
          <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 16, display: "flex", alignItems: "center", gap: 8 }}>
            <Clock size={18} style={{ color: "var(--color-accent)" }} />
            Heartbeat History
          </h3>
          {history.length > 0 ? (
            <div style={{ display: "flex", flexDirection: "column", gap: 8, maxHeight: 320, overflowY: "auto" }}>
              {history.map((h) => (
                <div key={h.id} style={{
                  display: "flex", alignItems: "center", gap: 10,
                  padding: "8px 12px", borderRadius: "var(--radius-sm)",
                  background: "var(--color-bg)", fontSize: "0.8rem",
                }}>
                  <CheckCircle2 size={14} style={{ color: "var(--color-success)", flexShrink: 0 }} />
                  <span style={{ flex: 1, color: "var(--color-text-secondary)" }}>
                    {new Date(h.occurredAt).toLocaleString()}
                  </span>
                  <span className="badge badge-neutral" style={{ fontSize: "0.65rem" }}>
                    {h.source}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className="empty-state" style={{ padding: 30 }}>
              <HeartPulse size={30} />
              <p>No heartbeats recorded yet</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between" }}>
      <span style={{ color: "var(--color-text-secondary)" }}>{label}</span>
      <span>{value}</span>
    </div>
  );
}

const pulseRingStyle: React.CSSProperties = {
  position: "absolute",
  top: "50%",
  left: "50%",
  transform: "translate(-50%, -50%)",
  width: 80,
  height: 80,
  borderRadius: "50%",
  border: "2px solid var(--color-success)",
  animation: "pulseRing 1.5s ease-out forwards",
  pointerEvents: "none",
};

// Inject global animation
if (typeof document !== "undefined" && !document.getElementById("pulse-keyframes")) {
  const style = document.createElement("style");
  style.id = "pulse-keyframes";
  style.textContent = `@keyframes pulseRing { 0% { width: 80px; height: 80px; opacity: 0.6; } 100% { width: 300px; height: 300px; opacity: 0; } }`;
  document.head.appendChild(style);
}
