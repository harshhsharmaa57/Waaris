import { NavLink, useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import {
  LayoutDashboard,
  ScrollText,
  Users,
  HeartPulse,
  ShieldCheck,
  Bell,
  ClipboardList,
  LogOut,
  User,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { useState } from "react";

const navItems = [
  { to: "/dashboard", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/will", icon: ScrollText, label: "Digital Will" },
  { to: "/trustees", icon: Users, label: "Trustees" },
  { to: "/heartbeat", icon: HeartPulse, label: "Heartbeat" },
  { to: "/verifications", icon: ShieldCheck, label: "Verifications" },
  { to: "/notifications", icon: Bell, label: "Notifications" },
  { to: "/audit", icon: ClipboardList, label: "Audit Log" },
];

export function Sidebar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [collapsed, setCollapsed] = useState(false);

  const handleLogout = async () => {
    await logout();
    navigate("/login");
  };

  return (
    <aside
      style={{
        width: collapsed ? 68 : 260,
        minHeight: "100vh",
        background: "var(--color-bg-secondary)",
        borderRight: "1px solid var(--color-border)",
        display: "flex",
        flexDirection: "column",
        transition: "width var(--transition-base)",
        position: "sticky",
        top: 0,
        flexShrink: 0,
        zIndex: 40,
      }}
    >
      {/* Logo */}
      <div
        style={{
          padding: collapsed ? "20px 12px" : "20px 20px",
          borderBottom: "1px solid var(--color-border)",
          display: "flex",
          alignItems: "center",
          justifyContent: collapsed ? "center" : "space-between",
          minHeight: 72,
        }}
      >
        {!collapsed && (
          <div>
            <h2
              style={{
                fontSize: "1.3rem",
                fontWeight: 800,
                background: "linear-gradient(135deg, var(--color-accent), var(--color-accent-light))",
                WebkitBackgroundClip: "text",
                WebkitTextFillColor: "transparent",
                letterSpacing: "-0.02em",
              }}
            >
              वारिस
            </h2>
            <span
              style={{
                fontSize: "0.65rem",
                color: "var(--color-text-muted)",
                textTransform: "uppercase",
                letterSpacing: "0.1em",
              }}
            >
              Digital Legacy Vault
            </span>
          </div>
        )}
        <button
          className="btn-ghost btn-icon"
          onClick={() => setCollapsed(!collapsed)}
          aria-label="Toggle sidebar"
          style={{ borderRadius: "var(--radius-sm)" }}
        >
          {collapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
        </button>
      </div>

      {/* Nav */}
      <nav style={{ flex: 1, padding: "12px 8px", display: "flex", flexDirection: "column", gap: 2 }}>
        {navItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            style={({ isActive }) => ({
              display: "flex",
              alignItems: "center",
              gap: 12,
              padding: collapsed ? "10px 0" : "10px 14px",
              justifyContent: collapsed ? "center" : "flex-start",
              borderRadius: "var(--radius-md)",
              textDecoration: "none",
              fontSize: "0.875rem",
              fontWeight: isActive ? 600 : 500,
              color: isActive ? "var(--color-accent)" : "var(--color-text-secondary)",
              background: isActive ? "var(--color-accent-dim)" : "transparent",
              transition: "all var(--transition-fast)",
            })}
            title={label}
          >
            <Icon size={18} />
            {!collapsed && label}
          </NavLink>
        ))}
      </nav>

      {/* User */}
      <div
        style={{
          padding: collapsed ? "16px 8px" : "16px",
          borderTop: "1px solid var(--color-border)",
          display: "flex",
          flexDirection: "column",
          gap: 8,
        }}
      >
        {!collapsed && (
          <NavLink
            to="/profile"
            style={{
              display: "flex",
              alignItems: "center",
              gap: 10,
              textDecoration: "none",
              padding: "8px 0",
            }}
          >
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: "50%",
                background: "linear-gradient(135deg, var(--color-accent), #d97706)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: "0.8rem",
                fontWeight: 700,
                color: "#0a0e1a",
              }}
            >
              {(user?.displayName?.[0] || user?.email?.[0] || "U").toUpperCase()}
            </div>
            <div style={{ overflow: "hidden" }}>
              <div
                style={{
                  fontSize: "0.8rem",
                  fontWeight: 600,
                  color: "var(--color-text)",
                  whiteSpace: "nowrap",
                  textOverflow: "ellipsis",
                  overflow: "hidden",
                  maxWidth: 140,
                }}
              >
                {user?.displayName || "User"}
              </div>
              <div
                style={{
                  fontSize: "0.7rem",
                  color: "var(--color-text-muted)",
                  whiteSpace: "nowrap",
                  textOverflow: "ellipsis",
                  overflow: "hidden",
                  maxWidth: 140,
                }}
              >
                {user?.email}
              </div>
            </div>
          </NavLink>
        )}
        {collapsed && (
          <NavLink
            to="/profile"
            style={{
              display: "flex",
              justifyContent: "center",
              textDecoration: "none",
            }}
            title="Profile"
          >
            <User size={18} style={{ color: "var(--color-text-secondary)" }} />
          </NavLink>
        )}
        <button
          onClick={handleLogout}
          className="btn btn-ghost btn-sm"
          style={{
            width: "100%",
            justifyContent: collapsed ? "center" : "flex-start",
            color: "var(--color-danger)",
          }}
          title="Log out"
        >
          <LogOut size={16} />
          {!collapsed && "Log out"}
        </button>
      </div>
    </aside>
  );
}
