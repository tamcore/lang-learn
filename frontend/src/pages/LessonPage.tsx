import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { api } from "../api/client";
import type { Lesson, Turn } from "../api/types";

function TurnBubble({ turn }: { turn: Turn }) {
  const [revealed, setRevealed] = useState(!turn.is_blurred);
  const isSystem = turn.speaker === "system";

  return (
    <div
      className={`turn-bubble ${isSystem ? "turn-system" : "turn-user"}`}
      onClick={() => !revealed && setRevealed(true)}
    >
      <div className={!revealed ? "turn-blurred" : ""}>{turn.text}</div>
      {revealed && turn.translation && (
        <div className="turn-translation">{turn.translation}</div>
      )}
    </div>
  );
}

export default function LessonPage() {
  const { id, seq } = useParams<{ id: string; seq: string }>();
  const [lesson, setLesson] = useState<Lesson | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id && seq) {
      api.getLesson(id, parseInt(seq))
        .then(setLesson)
        .catch(console.error)
        .finally(() => setLoading(false));
    }
  }, [id, seq]);

  if (loading) return <div className="container">Loading lesson...</div>;
  if (!lesson) return <div className="container">Lesson not found</div>;

  return (
    <div className="container">
      <Link to={`/courses/${id}`} style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>
        ← Back to course
      </Link>
      <h2 style={{ margin: "1rem 0" }}>{lesson.title}</h2>
      <div className="lesson-container">
        {lesson.turns.map((turn) => (
          <TurnBubble key={turn.id} turn={turn} />
        ))}
      </div>
    </div>
  );
}
