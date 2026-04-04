import { useEffect, useState, useCallback } from "react";
import { api } from "../api/client";
import type { User } from "../api/types";

interface JobInfo {
  id: string;
  courseId: string;
  status: string;
  progress: number;
}

export default function AdminPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [courses, setCourses] = useState<any[]>([]);
  const [tab, setTab] = useState<"users" | "courses" | "audit">("users");
  const [audit, setAudit] = useState<any[]>([]);

  // Create user form
  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newIsAdmin, setNewIsAdmin] = useState(false);
  const [userError, setUserError] = useState("");

  // Generate course form
  const [genSource, setGenSource] = useState("en");
  const [genTarget, setGenTarget] = useState("de");
  const [genDirection, setGenDirection] = useState("forward");
  const [genLessons, setGenLessons] = useState(10);
  const [genError, setGenError] = useState("");

  // Job tracking
  const [jobs, setJobs] = useState<JobInfo[]>([]);

  const refreshCourses = useCallback(() => {
    api.getAdminCourses().then(setCourses).catch(console.error);
  }, []);

  useEffect(() => {
    api.getUsers().then(setUsers).catch(console.error);
    refreshCourses();
  }, [refreshCourses]);

  // Poll active jobs
  useEffect(() => {
    const active = jobs.filter((j) => j.status === "running" || j.status === "pending");
    if (active.length === 0) return;

    const interval = setInterval(async () => {
      const updated = await Promise.all(
        jobs.map(async (j) => {
          if (j.status === "completed" || j.status === "failed") return j;
          try {
            const status = await api.getJobStatus(j.id);
            return { ...j, status: status.status, progress: status.progress };
          } catch {
            return j;
          }
        })
      );
      setJobs(updated);
      if (updated.some((j) => j.status === "completed")) refreshCourses();
    }, 3000);

    return () => clearInterval(interval);
  }, [jobs, refreshCourses]);

  const loadAudit = () => {
    api.getAudit().then(setAudit).catch(console.error);
  };

  const createUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setUserError("");
    try {
      await api.createUser(newUsername, newPassword, newIsAdmin);
      setNewUsername("");
      setNewPassword("");
      setNewIsAdmin(false);
      setUsers(await api.getUsers());
    } catch (err: any) {
      setUserError(err.message || "Failed to create user");
    }
  };

  const toggleAdmin = async (user: User) => {
    await api.updateUser(user.id, { is_admin: !user.is_admin });
    setUsers(await api.getUsers());
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

  const generateCourse = async (e: React.FormEvent) => {
    e.preventDefault();
    setGenError("");
    try {
      const result = await api.generateCourse(genSource, genTarget, genDirection, genLessons);
      setJobs((prev) => [...prev, { id: result.job_id, courseId: "", status: "running", progress: 0 }]);
    } catch (err: any) {
      setGenError(err.message || "Failed to start generation");
    }
  };

  const generateAudio = async (courseId: string) => {
    try {
      const result = await api.generateAudio(courseId);
      setJobs((prev) => [...prev, { id: result.job_id, courseId, status: "running", progress: 0 }]);
    } catch (err: any) {
      alert("Failed to start audio generation: " + (err.message || "unknown error"));
    }
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

      {/* Active Jobs Banner */}
      {jobs.filter((j) => j.status === "running" || j.status === "pending").length > 0 && (
        <div style={{ background: "var(--bg-card)", padding: "0.75rem", borderRadius: "8px", marginBottom: "1rem" }}>
          <strong>Active Jobs</strong>
          {jobs.filter((j) => j.status === "running" || j.status === "pending").map((j) => (
            <div key={j.id} style={{ marginTop: "0.25rem", fontSize: "0.85rem" }}>
              {j.id.slice(0, 20)}… — {j.status} ({Math.round(j.progress * 100)}%)
            </div>
          ))}
        </div>
      )}

      {tab === "users" && (
        <>
          <form onSubmit={createUser} style={{ display: "flex", gap: "0.5rem", marginBottom: "1rem", flexWrap: "wrap", alignItems: "center" }}>
            <input
              type="text" placeholder="Username" value={newUsername}
              onChange={(e) => setNewUsername(e.target.value)} required
              style={{ padding: "0.4rem 0.6rem", borderRadius: "6px", border: "1px solid var(--border)" }}
            />
            <input
              type="password" placeholder="Password" value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)} required
              style={{ padding: "0.4rem 0.6rem", borderRadius: "6px", border: "1px solid var(--border)" }}
            />
            <label style={{ display: "flex", alignItems: "center", gap: "0.25rem", fontSize: "0.85rem" }}>
              <input type="checkbox" checked={newIsAdmin} onChange={(e) => setNewIsAdmin(e.target.checked)} />
              Admin
            </label>
            <button type="submit" className="btn-primary">Create User</button>
          </form>
          {userError && <p style={{ color: "var(--danger)", marginBottom: "0.5rem" }}>{userError}</p>}

          <table>
            <thead>
              <tr><th>Username</th><th>Admin</th><th>Actions</th></tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td>{u.username}</td>
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
        </>
      )}

      {tab === "courses" && (
        <>
          <form onSubmit={generateCourse} style={{ display: "flex", gap: "0.5rem", marginBottom: "1rem", flexWrap: "wrap", alignItems: "center" }}>
            <input
              type="text" placeholder="Source (en)" value={genSource}
              onChange={(e) => setGenSource(e.target.value)} required
              style={{ width: "80px", padding: "0.4rem 0.6rem", borderRadius: "6px", border: "1px solid var(--border)" }}
            />
            <span>→</span>
            <input
              type="text" placeholder="Target (de)" value={genTarget}
              onChange={(e) => setGenTarget(e.target.value)} required
              style={{ width: "80px", padding: "0.4rem 0.6rem", borderRadius: "6px", border: "1px solid var(--border)" }}
            />
            <select
              value={genDirection} onChange={(e) => setGenDirection(e.target.value)}
              style={{ padding: "0.4rem 0.6rem", borderRadius: "6px", border: "1px solid var(--border)" }}
            >
              <option value="forward">Forward</option>
              <option value="reverse">Reverse</option>
            </select>
            <input
              type="number" min={1} max={30} value={genLessons}
              onChange={(e) => setGenLessons(Number(e.target.value))}
              style={{ width: "60px", padding: "0.4rem 0.6rem", borderRadius: "6px", border: "1px solid var(--border)" }}
            />
            <span style={{ fontSize: "0.85rem" }}>lessons</span>
            <button type="submit" className="btn-primary">Generate Course</button>
          </form>
          {genError && <p style={{ color: "var(--danger)", marginBottom: "0.5rem" }}>{genError}</p>}

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
                  <td style={{ display: "flex", gap: "0.5rem" }}>
                    <button className="btn-primary" onClick={() => generateAudio(c.id)}>🔊 Audio</button>
                    <button className="btn-danger" onClick={() => deleteCourse(c.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
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
