import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
import { api } from "../api/client";
import type { Lesson, Turn } from "../api/types";
import SpeakingFeedback from "../components/lesson/SpeakingFeedback";

function TurnBubble({
  turn,
  isActive,
  onAudioEnd,
}: {
  turn: Turn;
  isActive: boolean;
  onAudioEnd: () => void;
}) {
  const [revealed, setRevealed] = useState(!turn.is_blurred);
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const isSystem = turn.speaker === "system";

  useEffect(() => {
    if (isActive && isSystem && turn.audio_file) {
      const audio = new Audio(`/api/audio/${turn.audio_file}`);
      audioRef.current = audio;
      audio.onended = onAudioEnd;
      audio.onerror = onAudioEnd;
      audio.play().catch(onAudioEnd);
      return () => {
        audio.pause();
        audio.onended = null;
        audio.onerror = null;
      };
    }
    if (isActive && !isSystem) {
      if (turn.is_blurred && !revealed) setRevealed(true);
    }
  }, [isActive, isSystem, turn.audio_file, turn.is_blurred, onAudioEnd, revealed]);

  return (
    <div
      className={`turn-bubble ${isSystem ? "turn-system" : "turn-user"} ${isActive ? "turn-active" : ""}`}
      onClick={() => !revealed && setRevealed(true)}
    >
      <div className={!revealed ? "turn-blurred" : ""}>{turn.text}</div>
      {revealed && turn.translation && (
        <div className="turn-translation">{turn.translation}</div>
      )}
      {isActive && !isSystem && revealed && (
        <SpeakingFeedback
          expectedText={turn.text}
          onResult={() => onAudioEnd()}
        />
      )}
    </div>
  );
}

export default function LessonPage() {
  const { id, seq } = useParams<{ id: string; seq: string }>();
  const [lesson, setLesson] = useState<Lesson | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTurn, setActiveTurn] = useState(0);
  const [autoPlay, setAutoPlay] = useState(false);

  useEffect(() => {
    if (id && seq) {
      api.getLesson(id, parseInt(seq))
        .then(setLesson)
        .catch(console.error)
        .finally(() => setLoading(false));
    }
  }, [id, seq]);

  const advanceTurn = useCallback(() => {
    setActiveTurn((prev) => {
      if (lesson && prev < lesson.turns.length - 1) return prev + 1;
      return prev;
    });
  }, [lesson]);

  if (loading) return <div className="container">Loading lesson...</div>;
  if (!lesson) return <div className="container">Lesson not found</div>;

  return (
    <div className="container">
      <div style={{ display: "flex", alignItems: "center", gap: "1rem", marginBottom: "1rem" }}>
        <Link to={`/courses/${id}`} style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>
          ← Back to course
        </Link>
        <button
          className={`btn-primary ${autoPlay ? "btn-danger" : ""}`}
          onClick={() => { setAutoPlay(!autoPlay); if (!autoPlay) setActiveTurn(0); }}
          style={{ marginLeft: "auto", padding: "0.3rem 0.8rem", fontSize: "0.8rem" }}
        >
          {autoPlay ? "⏹ Stop" : "▶ Play Lesson"}
        </button>
      </div>
      <h2 style={{ margin: "0 0 1rem" }}>{lesson.title}</h2>
      <div className="lesson-container">
        {lesson.turns.map((turn, i) => (
          <TurnBubble
            key={turn.id}
            turn={turn}
            isActive={autoPlay && i === activeTurn}
            onAudioEnd={advanceTurn}
          />
        ))}
      </div>
    </div>
  );
}
