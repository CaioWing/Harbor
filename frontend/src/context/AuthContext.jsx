import { createContext, useContext, useMemo, useState } from "react";
import { loginManagement } from "../services/managementApi";

const TOKEN_KEY = "harbor_frontend_token";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem(TOKEN_KEY) || "");
  const [authLoading, setAuthLoading] = useState(false);
  const [authError, setAuthError] = useState("");

  async function signIn(credentials) {
    setAuthLoading(true);
    setAuthError("");

    try {
      const response = await loginManagement(credentials);
      localStorage.setItem(TOKEN_KEY, response.token);
      setToken(response.token);
    } catch (error) {
      setAuthError(error.message || "Falha de autenticacao");
      throw error;
    } finally {
      setAuthLoading(false);
    }
  }

  function signOut() {
    localStorage.removeItem(TOKEN_KEY);
    setToken("");
    setAuthError("");
  }

  const value = useMemo(
    () => ({
      token,
      isAuthenticated: Boolean(token),
      authLoading,
      authError,
      signIn,
      signOut,
      clearAuthError: () => setAuthError("")
    }),
    [token, authLoading, authError]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return context;
}
