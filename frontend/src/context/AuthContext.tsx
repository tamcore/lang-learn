import {
  createContext,
  useContext,
  useReducer,
  useEffect,
  type ReactNode,
} from "react";
import type { User } from "../api/types";
import { clearTokens } from "../api/client";

interface AuthState {
  user: User | null;
  loading: boolean;
}

type AuthAction =
  | { type: "SET_USER"; user: User }
  | { type: "LOGOUT" }
  | { type: "LOADED" };

const AuthContext = createContext<{
  state: AuthState;
  dispatch: React.Dispatch<AuthAction>;
}>({
  state: { user: null, loading: true },
  dispatch: () => {},
});

function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case "SET_USER":
      return { user: action.user, loading: false };
    case "LOGOUT":
      return { user: null, loading: false };
    case "LOADED":
      return { ...state, loading: false };
    default:
      return state;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(authReducer, {
    user: null,
    loading: true,
  });

  useEffect(() => {
    const stored = localStorage.getItem("user");
    if (stored) {
      try {
        dispatch({ type: "SET_USER", user: JSON.parse(stored) });
      } catch {
        dispatch({ type: "LOADED" });
      }
    } else {
      dispatch({ type: "LOADED" });
    }
  }, []);

  return (
    <AuthContext.Provider value={{ state, dispatch }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}

export function loginUser(
  dispatch: React.Dispatch<AuthAction>,
  user: User
) {
  localStorage.setItem("user", JSON.stringify(user));
  dispatch({ type: "SET_USER", user });
}

export function logoutUser(dispatch: React.Dispatch<AuthAction>) {
  clearTokens();
  localStorage.removeItem("user");
  dispatch({ type: "LOGOUT" });
}
