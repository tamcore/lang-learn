export interface User {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface Course {
  id: string;
  title: string;
  description: string;
  source_lang: string;
  target_lang: string;
  direction: "forward" | "reverse";
  perspective: "male" | "female";
  lesson_count: number;
}

export interface CourseFull extends Course {
  lessons: Lesson[];
}

export interface Lesson {
  id: string;
  course_id: string;
  sequence: number;
  title: string;
  turns: Turn[];
}

export interface Turn {
  id: string;
  sequence: number;
  speaker: "system" | "user";
  text: string;
  translation: string;
  audio_file: string;
  is_blurred: boolean;
  spaced_repeat: boolean;
  delay_after_ms: number;
}

export interface CourseProgress {
  user_id: string;
  course_id: string;
  current_lesson: number;
}

export interface ApiResponse<T> {
  data?: T;
  error?: string;
}
