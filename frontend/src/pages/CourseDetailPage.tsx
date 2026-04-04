import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { api } from "../api/client";
import type { CourseFull } from "../api/types";

export default function CourseDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [course, setCourse] = useState<CourseFull | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      api.getCourse(id).then(setCourse).catch(console.error).finally(() => setLoading(false));
    }
  }, [id]);

  if (loading) return <div className="container">Loading...</div>;
  if (!course) return <div className="container">Course not found</div>;

  return (
    <div className="container">
      <Link to="/" style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>← Back to courses</Link>
      <h2 style={{ margin: "1rem 0 0.5rem" }}>{course.title}</h2>
      <p style={{ color: "var(--text-muted)", marginBottom: "1rem" }}>{course.description}</p>
      <div className="grid">
        {course.lessons.map((l) => (
          <Link key={l.id} to={`/courses/${id}/lessons/${l.sequence}`} style={{ textDecoration: "none", color: "inherit" }}>
            <div className="card" style={{ cursor: "pointer" }}>
              <h3>Lesson {l.sequence}</h3>
              <p style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>{l.title}</p>
              <span className="badge badge-lang">{l.turns.length} turns</span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
