import { useState } from 'react'
import { useMediaRecorder } from '../../hooks/useMediaRecorder'

interface SpeakingFeedbackProps {
  expectedText: string
  onResult?: (score: number, transcript: string) => void
}

interface EvalResult {
  transcript: string
  score: number
  feedback: string
}

export default function SpeakingFeedback({ expectedText, onResult }: SpeakingFeedbackProps) {
  const { isRecording, audioBase64, startRecording, stopRecording, error: recError } = useMediaRecorder()
  const [result, setResult] = useState<EvalResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleRecord = async () => {
    if (isRecording) {
      await stopRecording()
    } else {
      setResult(null)
      setError(null)
      await startRecording()
    }
  }

  const handleSubmit = async () => {
    if (!audioBase64) return
    setLoading(true)
    setError(null)
    try {
      const token = localStorage.getItem('access_token') || sessionStorage.getItem('access_token')
      const resp = await fetch('/api/speaking/evaluate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ audio_base64: audioBase64, expected_text: expectedText }),
      })
      if (!resp.ok) throw new Error(`Evaluation failed: ${resp.status}`)
      const body = await resp.json()
      const evalResult = body.data as EvalResult
      setResult(evalResult)
      onResult?.(evalResult.score, evalResult.transcript)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Evaluation failed')
    } finally {
      setLoading(false)
    }
  }

  const scoreColor = (score: number) => {
    if (score >= 0.8) return 'var(--color-success, #22c55e)'
    if (score >= 0.5) return 'var(--color-warning, #eab308)'
    return 'var(--color-error, #ef4444)'
  }

  return (
    <div className="speaking-feedback">
      <div className="speaking-controls">
        <button
          className={`record-btn ${isRecording ? 'recording' : ''}`}
          onClick={handleRecord}
          disabled={loading}
        >
          {isRecording ? '⏹ Stop' : '🎤 Record'}
        </button>
        {audioBase64 && !isRecording && (
          <button className="submit-btn" onClick={handleSubmit} disabled={loading}>
            {loading ? 'Evaluating...' : 'Submit'}
          </button>
        )}
      </div>

      {(recError || error) && <p className="speaking-error">{recError || error}</p>}

      {result && (
        <div className="speaking-result">
          <div className="score-bar">
            <div
              className="score-fill"
              style={{ width: `${result.score * 100}%`, backgroundColor: scoreColor(result.score) }}
            />
          </div>
          <p className="score-text">{Math.round(result.score * 100)}%</p>
          <p className="feedback-text">{result.feedback}</p>
          {result.transcript && (
            <p className="transcript-text">
              You said: <em>{result.transcript}</em>
            </p>
          )}
        </div>
      )}
    </div>
  )
}
