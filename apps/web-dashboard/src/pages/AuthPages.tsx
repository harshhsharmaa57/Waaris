import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { ApiClientError } from "../lib/api";
import { Shield, Eye, EyeOff } from "lucide-react";

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(email, password);
      navigate("/dashboard");
    } catch (err) {
      setError(err instanceof ApiClientError ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={authPageStyle}>
      <div style={authBgGlow} />
      <div className="glass-card animate-in" style={authCardStyle}>
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <div style={logoIconStyle}>
            <Shield size={28} />
          </div>
          <h1 style={authTitleStyle}>Welcome back</h1>
          <p style={{ color: "var(--color-text-secondary)", fontSize: "0.9rem" }}>
            Sign in to your Waaris vault
          </p>
        </div>

        {error && <div style={errorStyle}>{error}</div>}

        <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: 18 }}>
          <div className="input-group">
            <label htmlFor="login-email">Email</label>
            <input
              id="login-email"
              className="input"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoFocus
            />
          </div>
          <div className="input-group">
            <label htmlFor="login-password">Password</label>
            <div style={{ position: "relative" }}>
              <input
                id="login-password"
                className="input"
                type={showPw ? "text" : "password"}
                placeholder="••••••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={12}
              />
              <button
                type="button"
                onClick={() => setShowPw(!showPw)}
                style={pwToggleStyle}
                tabIndex={-1}
              >
                {showPw ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </div>
          <button className="btn btn-primary btn-lg" type="submit" disabled={loading} style={{ width: "100%", marginTop: 8 }}>
            {loading ? <div className="spinner" style={{ borderTopColor: "#0a0e1a" }} /> : "Sign in"}
          </button>
        </form>

        <p style={{ textAlign: "center", marginTop: 24, fontSize: "0.85rem", color: "var(--color-text-secondary)" }}>
          Don't have an account?{" "}
          <Link to="/register" style={linkStyle}>
            Create one
          </Link>
        </p>
      </div>
    </div>
  );
}

export function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await register(email, password, name || undefined);
      navigate("/dashboard");
    } catch (err) {
      setError(err instanceof ApiClientError ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={authPageStyle}>
      <div style={authBgGlow} />
      <div className="glass-card animate-in" style={authCardStyle}>
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <div style={logoIconStyle}>
            <Shield size={28} />
          </div>
          <h1 style={authTitleStyle}>Create your vault</h1>
          <p style={{ color: "var(--color-text-secondary)", fontSize: "0.9rem" }}>
            Begin securing your digital legacy
          </p>
        </div>

        {error && <div style={errorStyle}>{error}</div>}

        <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: 18 }}>
          <div className="input-group">
            <label htmlFor="reg-name">Display name (optional)</label>
            <input
              id="reg-name"
              className="input"
              type="text"
              placeholder="Your name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={100}
            />
          </div>
          <div className="input-group">
            <label htmlFor="reg-email">Email</label>
            <input
              id="reg-email"
              className="input"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <div className="input-group">
            <label htmlFor="reg-password">Password (min 12 characters)</label>
            <div style={{ position: "relative" }}>
              <input
                id="reg-password"
                className="input"
                type={showPw ? "text" : "password"}
                placeholder="••••••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={12}
              />
              <button
                type="button"
                onClick={() => setShowPw(!showPw)}
                style={pwToggleStyle}
                tabIndex={-1}
              >
                {showPw ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </div>
          <button className="btn btn-primary btn-lg" type="submit" disabled={loading} style={{ width: "100%", marginTop: 8 }}>
            {loading ? <div className="spinner" style={{ borderTopColor: "#0a0e1a" }} /> : "Create account"}
          </button>
        </form>

        <p style={{ textAlign: "center", marginTop: 24, fontSize: "0.85rem", color: "var(--color-text-secondary)" }}>
          Already have an account?{" "}
          <Link to="/login" style={linkStyle}>
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}

// Shared styles
const authPageStyle: React.CSSProperties = {
  minHeight: "100vh",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  padding: 20,
  position: "relative",
  overflow: "hidden",
};

const authBgGlow: React.CSSProperties = {
  position: "absolute",
  top: "-40%",
  left: "50%",
  transform: "translateX(-50%)",
  width: 600,
  height: 600,
  borderRadius: "50%",
  background: "radial-gradient(circle, rgba(245,158,11,0.08) 0%, transparent 70%)",
  pointerEvents: "none",
};

const authCardStyle: React.CSSProperties = {
  width: "100%",
  maxWidth: 420,
  padding: 40,
  position: "relative",
  zIndex: 1,
};

const logoIconStyle: React.CSSProperties = {
  width: 56,
  height: 56,
  borderRadius: "var(--radius-lg)",
  background: "linear-gradient(135deg, var(--color-accent), #d97706)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  margin: "0 auto 16px",
  color: "#0a0e1a",
};

const authTitleStyle: React.CSSProperties = {
  fontSize: "1.5rem",
  fontWeight: 800,
  color: "var(--color-text)",
  marginBottom: 4,
};

const errorStyle: React.CSSProperties = {
  padding: "10px 14px",
  borderRadius: "var(--radius-md)",
  background: "var(--color-danger-dim)",
  color: "var(--color-danger)",
  fontSize: "0.85rem",
  marginBottom: 8,
  border: "1px solid rgba(239,68,68,0.2)",
};

const linkStyle: React.CSSProperties = {
  color: "var(--color-accent)",
  textDecoration: "none",
  fontWeight: 600,
};

const pwToggleStyle: React.CSSProperties = {
  position: "absolute",
  right: 10,
  top: "50%",
  transform: "translateY(-50%)",
  background: "none",
  border: "none",
  color: "var(--color-text-muted)",
  cursor: "pointer",
  padding: 4,
};
