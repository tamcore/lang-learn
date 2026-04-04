import { BrowserRouter, Routes, Route, Navigate, Link } from "react-router-dom";
import { AuthProvider, useAuth, logoutUser } from "./context/AuthContext";
import LoginPage from "./pages/LoginPage";
import CoursesPage from "./pages/CoursesPage";
import CourseDetailPage from "./pages/CourseDetailPage";
import LessonPage from "./pages/LessonPage";
import AdminPage from "./pages/AdminPage";
import { api } from "./api/client";
import "./styles/app.css";

function Nav() {
  const { state, dispatch } = useAuth();
  if (!state.user) return null;

  return (
    <nav className="nav">
      <h1><Link to="/" style={{ color: "inherit" }}>🎓 Lang Learn</Link></h1>
      {state.user.is_admin && <Link to="/admin">Admin</Link>}
      <span style={{ color: "var(--text-muted)" }}>{state.user.username}</span>
      <button
        className="btn-danger"
        onClick={() => { api.logout(); logoutUser(dispatch); }}
        style={{ padding: "0.3rem 0.6rem", fontSize: "0.8rem" }}
      >
        Logout
      </button>
    </nav>
  );
}

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { state } = useAuth();
  if (state.loading) return <div className="container">Loading...</div>;
  if (!state.user) return <Navigate to="/login" />;
  return <>{children}</>;
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { state } = useAuth();
  if (state.loading) return <div className="container">Loading...</div>;
  if (!state.user?.is_admin) return <Navigate to="/" />;
  return <>{children}</>;
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Nav />
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<PrivateRoute><CoursesPage /></PrivateRoute>} />
          <Route path="/courses/:id" element={<PrivateRoute><CourseDetailPage /></PrivateRoute>} />
          <Route path="/courses/:id/lessons/:seq" element={<PrivateRoute><LessonPage /></PrivateRoute>} />
          <Route path="/admin" element={<AdminRoute><AdminPage /></AdminRoute>} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
