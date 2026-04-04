import type { ApiResponse, TokenResponse } from "./types";

const BASE = "/api";

function getStorage(): Storage {
  return localStorage.getItem("remember_me") === "true"
    ? localStorage
    : sessionStorage;
}

function getToken(): string | null {
  return getStorage().getItem("access_token");
}

function setTokens(access: string, refresh?: string, remember?: boolean) {
  if (remember) {
    localStorage.setItem("remember_me", "true");
  }
  const store = getStorage();
  store.setItem("access_token", access);
  if (refresh) {
    store.setItem("refresh_token", refresh);
  }
}

export function clearTokens() {
  for (const s of [localStorage, sessionStorage]) {
    s.removeItem("access_token");
    s.removeItem("refresh_token");
    s.removeItem("user");
  }
  localStorage.removeItem("remember_me");
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE}${path}`, { ...options, headers });

  if (res.status === 401 && token) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      headers["Authorization"] = `Bearer ${getToken()}`;
      const retry = await fetch(`${BASE}${path}`, { ...options, headers });
      const body: ApiResponse<T> = await retry.json();
      if (body.error) throw new Error(body.error);
      return body.data as T;
    }
    clearTokens();
    window.location.href = "/login";
    throw new Error("Session expired");
  }

  const body: ApiResponse<T> = await res.json();
  if (body.error) throw new Error(body.error);
  return body.data as T;
}

async function tryRefresh(): Promise<boolean> {
  const refreshToken = getStorage().getItem("refresh_token");
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!res.ok) return false;
    const body: ApiResponse<TokenResponse> = await res.json();
    if (body.data) {
      setTokens(body.data.access_token, body.data.refresh_token);
      return true;
    }
    return false;
  } catch {
    return false;
  }
}

export const api = {
  login: async (username: string, password: string, rememberMe: boolean) => {
    const data = await request<TokenResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password, remember_me: rememberMe }),
    });
    setTokens(data.access_token, data.refresh_token, rememberMe);
    return data;
  },

  logout: async () => {
    try {
      await request("/auth/logout", { method: "POST" });
    } finally {
      clearTokens();
    }
  },

  getCourses: () => request<any[]>("/courses"),
  getCourse: (id: string) => request<any>(`/courses/${id}`),
  getLesson: (courseId: string, seq: number) =>
    request<any>(`/courses/${courseId}/lessons/${seq}`),

  getProgress: () => request<any[]>("/progress"),
  getCourseProgress: (courseId: string) =>
    request<any>(`/progress/${courseId}`),
  updateProgress: (courseId: string, currentLesson: number) =>
    request<any>(`/progress/${courseId}`, {
      method: "PUT",
      body: JSON.stringify({ current_lesson: currentLesson }),
    }),

  // Admin
  getUsers: () => request<any[]>("/admin/users"),
  createUser: (username: string, password: string, isAdmin: boolean) =>
    request<any>("/admin/users", {
      method: "POST",
      body: JSON.stringify({ username, password, is_admin: isAdmin }),
    }),
  updateUser: (id: string, data: any) =>
    request<any>(`/admin/users/${id}`, {
      method: "PATCH",
      body: JSON.stringify(data),
    }),
  deleteUser: (id: string) =>
    request<any>(`/admin/users/${id}`, { method: "DELETE" }),

  getAdminCourses: () => request<any[]>("/admin/courses"),
  deleteCourse: (id: string) =>
    request<any>(`/admin/courses/${id}`, { method: "DELETE" }),

  generateCourse: (sourceLang: string, targetLang: string, direction: string, lessonCount: number) =>
    request<any>("/admin/courses/generate", {
      method: "POST",
      body: JSON.stringify({ source_lang: sourceLang, target_lang: targetLang, direction, lesson_count: lessonCount }),
    }),
  generateAudio: (courseId: string) =>
    request<any>(`/admin/courses/${courseId}/audio`, { method: "POST" }),
  getJobStatus: (jobId: string) =>
    request<any>(`/admin/courses/generate/${jobId}`),

  getAudit: (date?: string) =>
    request<any[]>(`/admin/audit${date ? `?date=${date}` : ""}`),
};
