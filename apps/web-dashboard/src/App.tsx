import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Toaster } from "react-hot-toast";
import { AuthProvider } from "./contexts/AuthContext";
import { ProtectedRoute, PublicRoute } from "./components/RouteGuards";
import { LoginPage, RegisterPage } from "./pages/AuthPages";
import { DashboardPage } from "./pages/DashboardPage";
import { WillPage } from "./pages/WillPage";
import { TrusteesPage } from "./pages/TrusteesPage";
import { HeartbeatPage } from "./pages/HeartbeatPage";
import { VerificationsPage } from "./pages/VerificationsPage";
import { NotificationsPage } from "./pages/NotificationsPage";
import { AuditPage } from "./pages/AuditPage";
import { ProfilePage } from "./pages/ProfilePage";

export function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          {/* Public routes */}
          <Route element={<PublicRoute />}>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
          </Route>

          {/* Protected routes */}
          <Route element={<ProtectedRoute />}>
            <Route path="/dashboard" element={<DashboardPage />} />
            <Route path="/will" element={<WillPage />} />
            <Route path="/trustees" element={<TrusteesPage />} />
            <Route path="/heartbeat" element={<HeartbeatPage />} />
            <Route path="/verifications" element={<VerificationsPage />} />
            <Route path="/notifications" element={<NotificationsPage />} />
            <Route path="/audit" element={<AuditPage />} />
            <Route path="/profile" element={<ProfilePage />} />
          </Route>

          {/* Default redirect */}
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>

        <Toaster
          position="bottom-right"
          toastOptions={{
            style: {
              background: "#1e2744",
              color: "#e8ecf4",
              border: "1px solid rgba(99,115,171,0.2)",
              borderRadius: "10px",
              fontSize: "0.85rem",
            },
            success: {
              iconTheme: { primary: "#10b981", secondary: "#1e2744" },
            },
            error: {
              iconTheme: { primary: "#ef4444", secondary: "#1e2744" },
            },
          }}
        />
      </AuthProvider>
    </BrowserRouter>
  );
}
