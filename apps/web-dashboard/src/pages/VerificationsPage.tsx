import { useState, useEffect } from "react";
import { enrollment, type VerificationPending, ApiClientError } from "../lib/api";
import { ShieldCheck, CheckCircle2, XCircle, MinusCircle, Inbox } from "lucide-react";
import toast from "react-hot-toast";

export function VerificationsPage() {
  const [pending, setPending] = useState<VerificationPending[]>([]);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    try {
      const r = await enrollment.getPendingVerifications();
      setPending(r.pending ?? []);
    } catch { /* empty */ }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const act = async (id: string, action: "approve" | "reject" | "abstain") => {
    try {
      if (action === "approve") await enrollment.approve(id);
      else if (action === "reject") await enrollment.reject(id);
      else await enrollment.abstain(id);
      toast.success(`Verification ${action}d`);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiClientError ? err.message : `Failed to ${action}`);
    }
  };

  if (loading)
    return (
      <div className="page-container" style={{ display: "flex", justifyContent: "center", paddingTop: 80 }}>
        <div className="spinner spinner-lg" />
      </div>
    );

  return (
    <div className="page-container animate-in">
      <div className="page-header">
        <h1>Verifications</h1>
        <p>Pending trustee verification requests assigned to you</p>
      </div>

      {pending.length > 0 ? (
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          {pending.map((v) => (
            <div key={v.id} className="glass-card" style={{ padding: 24, display: "flex", alignItems: "center", gap: 20, flexWrap: "wrap" }}>
              <div style={{
                width: 44, height: 44, borderRadius: "var(--radius-md)",
                background: "var(--color-warning-dim)",
                display: "flex", alignItems: "center", justifyContent: "center",
              }}>
                <ShieldCheck size={22} style={{ color: "var(--color-warning)" }} />
              </div>
              <div style={{ flex: 1, minWidth: 200 }}>
                <div style={{ fontSize: "0.95rem", fontWeight: 600 }}>Verification Request</div>
                <div style={{ fontSize: "0.8rem", color: "var(--color-text-secondary)", fontFamily: "var(--font-mono)" }}>
                  {v.id.substring(0, 8)}…
                </div>
                <span className={`badge ${v.status === "pending" ? "badge-warning" : "badge-neutral"}`} style={{ marginTop: 4 }}>
                  {v.status}
                </span>
              </div>
              <div style={{ display: "flex", gap: 8 }}>
                <button className="btn btn-sm" style={{ background: "var(--color-success-dim)", color: "var(--color-success)", border: "1px solid rgba(16,185,129,0.2)" }} onClick={() => act(v.id, "approve")}>
                  <CheckCircle2 size={14} /> Approve
                </button>
                <button className="btn btn-danger btn-sm" onClick={() => act(v.id, "reject")}>
                  <XCircle size={14} /> Reject
                </button>
                <button className="btn btn-secondary btn-sm" onClick={() => act(v.id, "abstain")}>
                  <MinusCircle size={14} /> Abstain
                </button>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="glass-card empty-state">
          <Inbox size={40} />
          <h3>No pending verifications</h3>
          <p>You'll see verification requests here when a will enters pending verification state</p>
        </div>
      )}
    </div>
  );
}
