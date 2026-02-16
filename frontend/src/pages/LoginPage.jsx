import { useState } from "react";
import { useAuth } from "../context/AuthContext";

export default function LoginPage() {
  const { signIn, authLoading, authError, clearAuthError } = useAuth();
  const [email, setEmail] = useState("admin@harbor.local");
  const [password, setPassword] = useState("admin");

  async function handleSubmit(event) {
    event.preventDefault();
    clearAuthError();

    try {
      await signIn({ email, password });
    } catch {
      // Error is exposed via authError in context.
    }
  }

  return (
    <main className="login-screen">
      <section className="login-card">
        <p className="eyebrow">Harbor Frontend</p>
        <h1>Dashboard de Operacao</h1>
        <p className="subtitle">
          Gerencie devices, deployments e auditoria com visao consolidada.
        </p>

        <form className="login-form" onSubmit={handleSubmit}>
          <label>
            Email
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="admin@harbor.local"
              required
            />
          </label>

          <label>
            Senha
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="admin"
              required
            />
          </label>

          <button type="submit" disabled={authLoading}>
            {authLoading ? "Entrando..." : "Entrar"}
          </button>

          {authError ? <p className="form-error">{authError}</p> : null}
        </form>
      </section>
    </main>
  );
}
