import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api/client";
import { useAuth, loginUser } from "../context/AuthContext";

export default function LoginPage() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [rememberMe, setRememberMe] = useState(false);
  const [error, setError] = useState("");
  const { dispatch } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      const data = await api.login(username, password, rememberMe);
      loginUser(dispatch, data.user, rememberMe);
      navigate("/");
    } catch (err: any) {
      setError(err.message || "Login failed");
    }
  };

  return (
    <div className="container" style={{ maxWidth: 400, marginTop: "4rem" }}>
      <h2 style={{ marginBottom: "1.5rem" }}>Login</h2>
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Username</label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            required
          />
        </div>
        <div className="form-group">
          <label>Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
        </div>
        <label style={{ display: "flex", alignItems: "center", gap: "0.5rem", margin: "0.75rem 0" }}>
          <input
            type="checkbox"
            checked={rememberMe}
            onChange={(e) => setRememberMe(e.target.checked)}
          />
          Remember me
        </label>
        {error && <p className="error">{error}</p>}
        <button type="submit" className="btn-primary" style={{ width: "100%", marginTop: "0.5rem" }}>
          Login
        </button>
      </form>
    </div>
  );
}
