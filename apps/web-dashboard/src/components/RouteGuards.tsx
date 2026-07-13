import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { Sidebar } from "./Sidebar";

export function ProtectedRoute() {
  const { user, loading } = useAuth();
  if (loading)
    return (
      <div className="loading-screen">
        <div className="spinner" />
        <span style={{ color: "var(--color-text-secondary)" }}>Restoring session…</span>
      </div>
    );
  if (!user) return <Navigate to="/login" replace />;
  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      <Sidebar />
      <main style={{ flex: 1, overflow: "auto" }}>
        <Outlet />
      </main>
    </div>
  );
}

export function PublicRoute() {
  const { user, loading } = useAuth();
  if (loading)
    return (
      <div className="loading-screen">
        <div className="spinner" />
      </div>
    );
  if (user) return <Navigate to="/dashboard" replace />;
  return <Outlet />;
}
