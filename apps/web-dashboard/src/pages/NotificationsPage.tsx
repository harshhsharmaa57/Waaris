import { useState, useEffect } from "react";
import { enrollment, type Notification } from "../lib/api";
import { Bell, Mail, AlertCircle, CheckCircle2, XCircle } from "lucide-react";

export function NotificationsPage() {
  const [notifs, setNotifs] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    enrollment.getNotifications()
      .then((r) => setNotifs(r.history ?? []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading)
    return (
      <div className="page-container" style={{ display: "flex", justifyContent: "center", paddingTop: 80 }}>
        <div className="spinner spinner-lg" />
      </div>
    );

  const statusIcon = (s: string) => {
    if (s === "sent") return <CheckCircle2 size={16} style={{ color: "var(--color-success)" }} />;
    if (s === "failed") return <XCircle size={16} style={{ color: "var(--color-danger)" }} />;
    return <AlertCircle size={16} style={{ color: "var(--color-warning)" }} />;
  };

  return (
    <div className="page-container animate-in">
      <div className="page-header">
        <h1>Notifications</h1>
        <p>Email notification history for your will lifecycle events</p>
      </div>

      {notifs.length > 0 ? (
        <div className="glass-card" style={{ overflow: "hidden" }}>
          <div className="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Status</th>
                  <th>ID</th>
                  <th>Time</th>
                </tr>
              </thead>
              <tbody>
                {notifs.map((n) => (
                  <tr key={n.id}>
                    <td>
                      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                        {statusIcon(n.status)}
                        <span className={`badge ${n.status === "sent" ? "badge-success" : n.status === "failed" ? "badge-danger" : "badge-warning"}`}>
                          {n.status}
                        </span>
                      </div>
                    </td>
                    <td style={{ fontFamily: "var(--font-mono)", fontSize: "0.75rem" }}>
                      {n.id.substring(0, 12)}…
                    </td>
                    <td>{new Date(n.createdAt).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="glass-card empty-state">
          <Mail size={40} />
          <h3>No notifications yet</h3>
          <p>Notifications appear when lifecycle events trigger email alerts</p>
        </div>
      )}
    </div>
  );
}
