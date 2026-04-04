import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { User } from "../api/types";

export default function AdminPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [courses, setCourses] = useState<any[]>([]);
  const [tab, setTab] = useState<"users" | "courses" | "audit">("users");
  const [audit, setAudit] = useState<any[]>([]);

  useEffect(() => {
    api.getUsers().then(setUsers).catch(console.error);
    api.getAdminCourses().then(setCourses).catch(console.error);
  }, []);

  const loadAudit = () => {
    api.getAudit().then(setAudit).catch(console.error);
  };

  const toggleAdmin = async (user: User) => {
    await api.updateUser(user.id, { is_admin: !user.is_admin });
    const updated = await api.getUsers();
    setUsers(updated);
  };

  const deleteUser = async (id: string) => {
    if (!confirm("Delete this user?")) return;
    await api.deleteUser(id);
    setUsers(users.filter((u) => u.id !== id));
  };

  const deleteCourse = async (id: string) => {
    if (!confirm("Delete this course?")) return;
    await api.deleteCourse(id);
    setCourses(courses.filter((c) => c.id !== id));
  };

  return (
    <div className="container">
      <h2 style={{ marginBottom: "1rem" }}>Admin Dashboard</h2>
      <div style={{ display: "flex", gap: "0.5rem", marginBottom: "1rem" }}>
        {(["users", "courses", "audit"] as const).map((t) => (
          <button
            key={t}
            className={tab === t ? "btn-primary" : ""}
            style={tab !== t ? { background: "var(--bg-card)", color: "var(--text)" } : {}}
            onClick={() => { setTab(t); if (t === "audit") loadAudit(); }}
          >
            {t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {tab === "users" && (
        <table>
          <thead>
            <tr><th>Username</th><th>Email</th><th>Admin</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id}>
                <td>{u.username}</td>
                <td>{u.email}</td>
                <td>{u.is_admin ? <span className="badge badge-admin">Admin</span> : "—"}</td>
                <td style={{ display: "flex", gap: "0.5rem" }}>
                  <button className="btn-primary" onClick={() => toggleAdmin(u)}>
                    {u.is_admin ? "Revoke" : "Make Admin"}
                  </button>
                  <button className="btn-danger" onClick={() => deleteUser(u.id)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {tab === "courses" && (
        <table>
          <thead>
            <tr><th>Title</th><th>Languages</th><th>Direction</th><th>Lessons</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {courses.map((c) => (
              <tr key={c.id}>
                <td>{c.title}</td>
                <td>{c.source_lang} → {c.target_lang}</td>
                <td>{c.direction}</td>
                <td>{c.lesson_count}</td>
                <td><button className="btn-danger" onClick={() => deleteCourse(c.id)}>Delete</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {tab === "audit" && (
        <table>
          <thead>
            <tr><th>Time</th><th>Action</th><th>Actor</th><th>Target</th></tr>
          </thead>
          <tbody>
            {audit.length === 0 ? (
              <tr><td colSpan={4} style={{ textAlign: "center", color: "var(--text-muted)" }}>No entries today</td></tr>
            ) : (
              audit.map((e, i) => (
                <tr key={i}>
                  <td>{new Date(e.timestamp).toLocaleTimeString()}</td>
                  <td>{e.action}</td>
                  <td>{e.actor_id}</td>
                  <td>{e.target_id}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      )}
    </div>
  );
}
