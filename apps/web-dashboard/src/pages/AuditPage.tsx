import { useState, useEffect } from "react";
import { enrollment, type AuditEvent } from "../lib/api";
import { ClipboardList, Search } from "lucide-react";

export function AuditPage() {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("");

  useEffect(() => {
    enrollment.getAuditHistory()
      .then((r) => setEvents(r.history ?? []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const filtered = events.filter((e) =>
    filter === "" ||
    e.event.toLowerCase().includes(filter.toLowerCase()) ||
    e.actor.toLowerCase().includes(filter.toLowerCase()) ||
    e.details.toLowerCase().includes(filter.toLowerCase()),
  );

  if (loading)
    return (
      <div className="page-container" style={{ display: "flex", justifyContent: "center", paddingTop: 80 }}>
        <div className="spinner spinner-lg" />
      </div>
    );

  return (
    <div className="page-container animate-in">
      <div className="page-header">
        <h1>Audit Log</h1>
        <p>Immutable record of all account and workflow events</p>
      </div>

      {events.length > 0 && (
        <div style={{ marginBottom: 20, position: "relative" }}>
          <Search size={16} style={{ position: "absolute", left: 14, top: "50%", transform: "translateY(-50%)", color: "var(--color-text-muted)" }} />
          <input
            className="input"
            placeholder="Filter events…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            style={{ paddingLeft: 38, maxWidth: 400 }}
          />
        </div>
      )}

      {filtered.length > 0 ? (
        <div className="glass-card" style={{ overflow: "hidden" }}>
          <div className="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Event</th>
                  <th>Actor</th>
                  <th>Details</th>
                  <th>Correlation</th>
                  <th>Time</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((e) => (
                  <tr key={e.id}>
                    <td>
                      <span className="badge badge-info" style={{ fontSize: "0.7rem" }}>
                        {e.event}
                      </span>
                    </td>
                    <td style={{ fontSize: "0.8rem" }}>{e.actor}</td>
                    <td style={{ maxWidth: 280, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", fontSize: "0.8rem" }}>
                      {e.details}
                    </td>
                    <td style={{ fontFamily: "var(--font-mono)", fontSize: "0.7rem" }}>
                      {e.correlationId?.substring(0, 8)}…
                    </td>
                    <td style={{ fontSize: "0.8rem", whiteSpace: "nowrap" }}>
                      {new Date(e.createdAt).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="glass-card empty-state">
          <ClipboardList size={40} />
          <h3>{filter ? "No matching events" : "No audit events"}</h3>
          <p>Events appear as you interact with your digital will</p>
        </div>
      )}
    </div>
  );
}
