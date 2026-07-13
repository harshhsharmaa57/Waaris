import { useState, useEffect, type FormEvent } from "react";
import { enrollment, type Will, ApiClientError } from "../lib/api";
import { ScrollText, Save, Trash2, History, Plus } from "lucide-react";
import toast from "react-hot-toast";

const CATEGORIES = ["financial", "private", "community_shareable"] as const;

export function WillPage() {
  const [will, setWill] = useState<Will | null>(null);
  const [history, setHistory] = useState<Will[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [form, setForm] = useState({
    state: "draft" as "draft" | "published",
    dormancyPeriodDays: 180,
    gracePeriodDays: 30,
    policyVersionAccepted: "2026-07",
    releaseCategories: ["financial", "private"] as string[],
  });

  useEffect(() => {
    enrollment
      .getWill()
      .then((w) => {
        setWill(w);
        setForm({
          state: w.state,
          dormancyPeriodDays: w.dormancyPeriodDays,
          gracePeriodDays: w.gracePeriodDays,
          policyVersionAccepted: w.policyVersionAccepted,
          releaseCategories: w.releaseCategories,
        });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const loadHistory = async () => {
    try {
      const h = await enrollment.getHistory();
      setHistory(h.history ?? []);
      setShowHistory(true);
    } catch { /* empty */ }
  };

  const toggleCategory = (cat: string) => {
    setForm((f) => ({
      ...f,
      releaseCategories: f.releaseCategories.includes(cat)
        ? f.releaseCategories.filter((c) => c !== cat)
        : [...f.releaseCategories, cat],
    }));
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (form.releaseCategories.length === 0) {
      toast.error("Select at least one release category");
      return;
    }
    setSaving(true);
    try {
      if (will) {
        const updated = await enrollment.updateWill(form);
        setWill(updated);
        toast.success("Will updated");
      } else {
        const created = await enrollment.createWill(form);
        setWill(created);
        toast.success("Will created");
      }
    } catch (err) {
      toast.error(err instanceof ApiClientError ? err.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete your active will?")) return;
    try {
      await enrollment.deleteWill();
      setWill(null);
      setForm({
        state: "draft",
        dormancyPeriodDays: 180,
        gracePeriodDays: 30,
        policyVersionAccepted: "2026-07",
        releaseCategories: ["financial", "private"],
      });
      toast.success("Will deleted");
    } catch (err) {
      toast.error(err instanceof ApiClientError ? err.message : "Failed to delete");
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
        <h1>Digital Will</h1>
        <p>Manage your legacy metadata, dormancy policy, and release categories</p>
      </div>

      <div className="grid-2">
        {/* Form */}
        <div className="glass-card" style={{ padding: 28 }}>
          <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 20, display: "flex", alignItems: "center", gap: 8 }}>
            <ScrollText size={18} style={{ color: "var(--color-accent)" }} />
            {will ? "Edit Will" : "Create Will"}
          </h3>

          <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: 18 }}>
            <div className="input-group">
              <label htmlFor="will-state">State</label>
              <select id="will-state" className="input" value={form.state} onChange={(e) => setForm({ ...form, state: e.target.value as "draft" | "published" })}>
                <option value="draft">Draft</option>
                <option value="published">Published</option>
              </select>
            </div>
            <div className="input-group">
              <label htmlFor="will-dormancy">Dormancy Period (days)</label>
              <input id="will-dormancy" className="input" type="number" min={1} max={3650} value={form.dormancyPeriodDays} onChange={(e) => setForm({ ...form, dormancyPeriodDays: Number(e.target.value) })} />
            </div>
            <div className="input-group">
              <label htmlFor="will-grace">Grace Period (days)</label>
              <input id="will-grace" className="input" type="number" min={1} max={365} value={form.gracePeriodDays} onChange={(e) => setForm({ ...form, gracePeriodDays: Number(e.target.value) })} />
            </div>
            <div className="input-group">
              <label htmlFor="will-policy">Policy Version</label>
              <input id="will-policy" className="input" type="text" maxLength={64} value={form.policyVersionAccepted} onChange={(e) => setForm({ ...form, policyVersionAccepted: e.target.value })} />
            </div>
            <div className="input-group">
              <label>Release Categories</label>
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                {CATEGORIES.map((cat) => (
                  <button
                    key={cat}
                    type="button"
                    onClick={() => toggleCategory(cat)}
                    className={`btn btn-sm ${form.releaseCategories.includes(cat) ? "btn-primary" : "btn-secondary"}`}
                  >
                    {cat.replace("_", " ")}
                  </button>
                ))}
              </div>
            </div>
            <div style={{ display: "flex", gap: 10, marginTop: 8 }}>
              <button className="btn btn-primary" type="submit" disabled={saving} style={{ flex: 1 }}>
                {saving ? <div className="spinner" style={{ borderTopColor: "#0a0e1a" }} /> : <><Save size={16} /> {will ? "Save" : "Create"}</>}
              </button>
              {will && (
                <button className="btn btn-danger" type="button" onClick={handleDelete}>
                  <Trash2 size={16} />
                </button>
              )}
            </div>
          </form>
        </div>

        {/* Info + History */}
        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          {will && (
            <div className="glass-card" style={{ padding: 24 }}>
              <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 16 }}>Current Will</h3>
              <div style={{ display: "flex", flexDirection: "column", gap: 10, fontSize: "0.85rem" }}>
                <Row label="ID" value={will.id} mono />
                <Row label="Version" value={`v${will.version}`} />
                <Row label="State" value={<span className={`badge ${will.state === "published" ? "badge-success" : "badge-neutral"}`}>{will.state}</span>} />
                <Row label="Created" value={new Date(will.createdAt).toLocaleDateString()} />
                <Row label="Updated" value={new Date(will.updatedAt).toLocaleDateString()} />
              </div>
              <button className="btn btn-ghost btn-sm" style={{ marginTop: 16, width: "100%" }} onClick={loadHistory}>
                <History size={14} /> View version history
              </button>
            </div>
          )}

          {!will && (
            <div className="glass-card empty-state">
              <Plus size={40} />
              <h3>No active will</h3>
              <p>Create your Digital Will to begin securing your legacy</p>
            </div>
          )}

          {showHistory && history.length > 0 && (
            <div className="glass-card" style={{ padding: 24 }}>
              <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 16 }}>
                <History size={18} style={{ color: "var(--color-accent)", verticalAlign: "middle", marginRight: 6 }} />
                Version History
              </h3>
              <div className="table-wrapper">
                <table>
                  <thead>
                    <tr>
                      <th>Ver</th>
                      <th>State</th>
                      <th>Dormancy</th>
                      <th>Grace</th>
                      <th>Created</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.map((v) => (
                      <tr key={v.id}>
                        <td style={{ fontWeight: 600 }}>v{v.version}</td>
                        <td><span className={`badge ${v.state === "published" ? "badge-success" : "badge-neutral"}`}>{v.state}</span></td>
                        <td>{v.dormancyPeriodDays}d</td>
                        <td>{v.gracePeriodDays}d</td>
                        <td>{new Date(v.createdAt).toLocaleDateString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function Row({ label, value, mono }: { label: string; value: React.ReactNode; mono?: boolean }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
      <span style={{ color: "var(--color-text-secondary)" }}>{label}</span>
      <span style={{ fontFamily: mono ? "var(--font-mono)" : undefined, fontSize: mono ? "0.75rem" : undefined, maxWidth: 180, overflow: "hidden", textOverflow: "ellipsis" }}>
        {value}
      </span>
    </div>
  );
}
