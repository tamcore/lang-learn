import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { Course } from "../api/types";

export default function CoursesPage() {
  const [courses, setCourses] = useState<Course[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getCourses().then(setCourses).catch(console.error).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="container">Loading courses...</div>;

  const langFlag: Record<string, string> = {
    sk: "🇸🇰", en: "🇬🇧", de: "🇩🇪", es: "🇪🇸", fr: "🇫🇷", it: "🇮🇹",
  };

  return (
    <div className="container">
      <h2 style={{ marginBottom: "1rem" }}>Courses</h2>
      {courses.length === 0 ? (
        <p style={{ color: "var(--text-muted)" }}>No courses available yet.</p>
      ) : (
        <div className="grid">
          {courses.map((c) => (
            <Link key={c.id} to={`/courses/${c.id}`} style={{ textDecoration: "none", color: "inherit" }}>
              <div className="card" style={{ cursor: "pointer" }}>
                <div style={{ fontSize: "1.5rem", marginBottom: "0.5rem" }}>
                  {langFlag[c.source_lang] || c.source_lang} → {langFlag[c.target_lang] || c.target_lang}
                </div>
                <h3>{c.title}</h3>
                <p style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>{c.description}</p>
                <div style={{ marginTop: "0.5rem", display: "flex", gap: "0.5rem" }}>
                  <span className="badge badge-lang">{c.direction}</span>
                  <span className="badge badge-lang">{c.lesson_count} lessons</span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
