import { useState, useEffect, type FormEvent } from "react";
import { enrollment, type Trustee, ApiClientError } from "../lib/api";
import { Users, Plus, Pencil, Trash2, X, UserPlus } from "lucide-react";
import toast from "react-hot-toast";

export function TrusteesPage() {
  const [trustees, setTrustees] = useState<Trustee[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Trustee | null>(null);
  const [form, setForm] = useState({ name: "", email: "", relationship: "" });
  const [saving, setSaving] = useState(false);

  const load = async () => {
    try {
      const r = await enrollment.listTrustees();
      setTrustees(r.trustees ?? []);
    } catch { /* empty */ }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const resetForm = () => {
    setForm({ name: "", email: "", relationship: "" });
    setEditing(null);
    setShowForm(false);
  };

  const handleEdit = (t: Trustee) => {
    setEditing(t);
    setForm({ name: t.name, email: t.email, relationship: t.relationship });
    setShowForm(true);
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      if (editing) {
        await enrollment.updateTrustee(editing.id, form);
        toast.success("Trustee updated");
      } else {
        await enrollment.addTrustee(form);
        toast.success("Trustee added");
      }
      resetForm();
      await load();
    } catch (err) {
      toast.error(err instanceof ApiClientError ? err.message : "Failed to save trustee");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Remove this trustee?")) return;
    try {
      await enrollment.deleteTrustee(id);
      toast.success("Trustee removed");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiClientError ? err.message : "Failed to remove trustee");
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
      <div className="page-header" style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
        <div>
          <h1>Trustees</h1>
          <p>People who hold key shares and verify your inactivity</p>
        </div>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            <Plus size={16} /> Add Trustee
          </button>
        )}
      </div>

      {/* Add/Edit Form */}
      {showForm && (
        <div className="glass-card animate-in" style={{ padding: 24, marginBottom: 24 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 18 }}>
            <h3 style={{ fontSize: "1rem", fontWeight: 700, display: "flex", alignItems: "center", gap: 8 }}>
              <UserPlus size={18} style={{ color: "var(--color-accent)" }} />
              {editing ? "Edit Trustee" : "Add Trustee"}
            </h3>
            <button className="btn btn-ghost btn-icon" onClick={resetForm}><X size={16} /></button>
          </div>
          <form onSubmit={handleSubmit} style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", gap: 16 }}>
            <div className="input-group">
              <label htmlFor="trustee-name">Name</label>
              <input id="trustee-name" className="input" type="text" maxLength={100} required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Full name" />
            </div>
            <div className="input-group">
              <label htmlFor="trustee-email">Email</label>
              <input id="trustee-email" className="input" type="email" maxLength={254} required value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} placeholder="trustee@example.com" />
            </div>
            <div className="input-group">
              <label htmlFor="trustee-rel">Relationship</label>
              <input id="trustee-rel" className="input" type="text" maxLength={100} required value={form.relationship} onChange={(e) => setForm({ ...form, relationship: e.target.value })} placeholder="e.g. Brother, Friend" />
            </div>
            <div style={{ display: "flex", alignItems: "flex-end", gap: 8 }}>
              <button className="btn btn-primary" type="submit" disabled={saving} style={{ flex: 1 }}>
                {saving ? <div className="spinner" style={{ borderTopColor: "#0a0e1a" }} /> : editing ? "Update" : "Add"}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Trustee Grid */}
      {trustees.length > 0 ? (
        <div className="grid-3">
          {trustees.map((t) => (
            <div key={t.id} className="glass-card" style={{ padding: 20 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 12 }}>
                <div style={{
                  width: 40, height: 40, borderRadius: "50%",
                  background: "linear-gradient(135deg, var(--color-info-dim), var(--color-accent-dim))",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  fontSize: "0.85rem", fontWeight: 700, color: "var(--color-accent)",
                }}>
                  {t.name[0].toUpperCase()}
                </div>
                <div style={{ display: "flex", gap: 4 }}>
                  <button className="btn btn-ghost btn-icon btn-sm" onClick={() => handleEdit(t)} title="Edit">
                    <Pencil size={14} />
                  </button>
                  <button className="btn btn-ghost btn-icon btn-sm" onClick={() => handleDelete(t.id)} title="Delete" style={{ color: "var(--color-danger)" }}>
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
              <div style={{ fontSize: "0.95rem", fontWeight: 600, marginBottom: 2 }}>{t.name}</div>
              <div style={{ fontSize: "0.8rem", color: "var(--color-text-secondary)", marginBottom: 4 }}>{t.email}</div>
              <span className="badge badge-neutral">{t.relationship}</span>
            </div>
          ))}
        </div>
      ) : (
        <div className="glass-card empty-state">
          <Users size={40} />
          <h3>No trustees configured</h3>
          <p>Add trusted individuals who will verify your inactivity</p>
        </div>
      )}
    </div>
  );
}
