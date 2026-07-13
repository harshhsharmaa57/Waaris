import { useState, type FormEvent } from "react";
import { useAuth } from "../contexts/AuthContext";
import { ApiClientError, auth as authApi } from "../lib/api";
import { User, Save, Trash2, Shield, Mail, Calendar } from "lucide-react";
import toast from "react-hot-toast";
import { useNavigate } from "react-router-dom";

export function ProfilePage() {
  const { user, updateProfile, logout } = useAuth();
  const navigate = useNavigate();
  const [name, setName] = useState(user?.displayName || "");
  const [saving, setSaving] = useState(false);

  const handleSave = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await updateProfile(name);
      toast.success("Profile updated");
    } catch (err) {
      toast.error(err instanceof ApiClientError ? err.message : "Failed to update");
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteAccount = async () => {
    if (!confirm("This will permanently delete your account and all associated data. Are you sure?")) return;
    if (!confirm("This action CANNOT be undone. Final confirmation?")) return;
    try {
      await authApi.deleteMe();
      await logout();
      navigate("/login");
      toast.success("Account deleted");
    } catch (err) {
      toast.error(err instanceof ApiClientError ? err.message : "Failed to delete account");
    }
  };

  return (
    <div className="page-container animate-in">
      <div className="page-header">
        <h1>Profile</h1>
        <p>Manage your account settings</p>
      </div>

      <div className="grid-2">
        {/* Profile Card */}
        <div className="glass-card" style={{ padding: 28 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 16, marginBottom: 28 }}>
            <div style={{
              width: 56, height: 56, borderRadius: "50%",
              background: "linear-gradient(135deg, var(--color-accent), #d97706)",
              display: "flex", alignItems: "center", justifyContent: "center",
              fontSize: "1.4rem", fontWeight: 800, color: "#0a0e1a",
            }}>
              {(user?.displayName?.[0] || user?.email?.[0] || "U").toUpperCase()}
            </div>
            <div>
              <div style={{ fontSize: "1.1rem", fontWeight: 700 }}>{user?.displayName || "User"}</div>
              <div style={{ fontSize: "0.8rem", color: "var(--color-text-secondary)" }}>{user?.email}</div>
            </div>
          </div>

          <form onSubmit={handleSave} style={{ display: "flex", flexDirection: "column", gap: 18 }}>
            <div className="input-group">
              <label htmlFor="profile-name">Display Name</label>
              <input
                id="profile-name"
                className="input"
                type="text"
                maxLength={100}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Your display name"
              />
            </div>
            <button className="btn btn-primary" type="submit" disabled={saving}>
              {saving ? <div className="spinner" style={{ borderTopColor: "#0a0e1a" }} /> : <><Save size={16} /> Save Changes</>}
            </button>
          </form>
        </div>

        {/* Account Info */}
        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          <div className="glass-card" style={{ padding: 24 }}>
            <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 16, display: "flex", alignItems: "center", gap: 8 }}>
              <Shield size={18} style={{ color: "var(--color-accent)" }} />
              Account Details
            </h3>
            <div style={{ display: "flex", flexDirection: "column", gap: 12, fontSize: "0.85rem" }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <span style={{ color: "var(--color-text-secondary)", display: "flex", alignItems: "center", gap: 6 }}>
                  <User size={14} /> User ID
                </span>
                <span style={{ fontFamily: "var(--font-mono)", fontSize: "0.75rem" }}>
                  {user?.id?.substring(0, 12)}…
                </span>
              </div>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <span style={{ color: "var(--color-text-secondary)", display: "flex", alignItems: "center", gap: 6 }}>
                  <Mail size={14} /> Email
                </span>
                <span>{user?.email}</span>
              </div>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <span style={{ color: "var(--color-text-secondary)", display: "flex", alignItems: "center", gap: 6 }}>
                  <Calendar size={14} /> Joined
                </span>
                <span>{user?.createdAt ? new Date(user.createdAt).toLocaleDateString() : "—"}</span>
              </div>
            </div>
          </div>

          {/* Danger Zone */}
          <div className="glass-card" style={{ padding: 24, borderColor: "rgba(239,68,68,0.2)" }}>
            <h3 style={{ fontSize: "1rem", fontWeight: 700, marginBottom: 8, color: "var(--color-danger)" }}>
              Danger Zone
            </h3>
            <p style={{ fontSize: "0.8rem", color: "var(--color-text-secondary)", marginBottom: 16 }}>
              Permanently delete your account, will, and all associated data. This cannot be undone.
            </p>
            <button className="btn btn-danger" onClick={handleDeleteAccount}>
              <Trash2 size={16} /> Delete Account
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
