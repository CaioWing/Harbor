import { useEffect, useMemo, useState } from "react";
import { fetchDashboardSnapshot, loginManagement } from "./api";

const TOKEN_KEY = "harbor_dashboard_token";

function formatDate(value) {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

function statusTone(status) {
  const current = (status || "").toLowerCase();
  if (["success", "completed", "accepted", "active"].includes(current)) {
    return "pill pill--positive";
  }
  if (["failure", "rejected", "cancelled"].includes(current)) {
    return "pill pill--negative";
  }
  if (["pending", "scheduled", "downloading", "installing"].includes(current)) {
    return "pill pill--warning";
  }
  return "pill";
}

function StatCard({ label, value, hint }) {
  return (
    <article className="stat-card">
      <p className="stat-card__label">{label}</p>
      <p className="stat-card__value">{value}</p>
      <p className="stat-card__hint">{hint}</p>
    </article>
  );
}

function LoginView({ onLogin, loading, error }) {
  const [email, setEmail] = useState("admin@harbor.local");
  const [password, setPassword] = useState("admin");

  async function handleSubmit(event) {
    event.preventDefault();
    onLogin({ email, password });
  }

  return (
    <main className="login-screen">
      <section className="login-card">
        <p className="eyebrow">Harbor Control Plane</p>
        <h1>Dashboard de Operacao</h1>
        <p className="subtitle">
          Acompanhe devices, deploys e auditoria em uma unica tela.
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

          <button type="submit" disabled={loading}>
            {loading ? "Entrando..." : "Entrar"}
          </button>

          {error ? <p className="form-error">{error}</p> : null}
        </form>
      </section>
    </main>
  );
}

export default function App() {
  const [token, setToken] = useState(() => localStorage.getItem(TOKEN_KEY) || "");
  const [snapshot, setSnapshot] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const stats = useMemo(() => {
    if (!snapshot) {
      return {
        devicesTotal: 0,
        devicesPending: 0,
        devicesAccepted: 0,
        devicesRejected: 0,
        deployTotal: 0,
        deployActive: 0,
        deployCompleted: 0,
        deployCancelled: 0
      };
    }

    const counts = snapshot.deviceCounts || {};
    const deploy = snapshot.deploymentStats || {};

    return {
      devicesTotal:
        Number(counts.pending || 0) +
        Number(counts.accepted || 0) +
        Number(counts.rejected || 0) +
        Number(counts.decommissioned || 0),
      devicesPending: Number(counts.pending || 0),
      devicesAccepted: Number(counts.accepted || 0),
      devicesRejected: Number(counts.rejected || 0),
      deployTotal: Number(deploy.total || 0),
      deployActive: Number(deploy.active || 0),
      deployCompleted: Number(deploy.completed || 0),
      deployCancelled: Number(deploy.cancelled || 0)
    };
  }, [snapshot]);

  async function loadDashboard(activeToken = token) {
    if (!activeToken) {
      return;
    }

    setLoading(true);
    setError("");

    try {
      const payload = await fetchDashboardSnapshot(activeToken);
      setSnapshot(payload);
    } catch (loadError) {
      setError(loadError.message);
      if (loadError.message.toLowerCase().includes("token")) {
        localStorage.removeItem(TOKEN_KEY);
        setToken("");
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleLogin(credentials) {
    setLoading(true);
    setError("");

    try {
      const response = await loginManagement(credentials);
      localStorage.setItem(TOKEN_KEY, response.token);
      setToken(response.token);
    } catch (loginError) {
      setError(loginError.message);
    } finally {
      setLoading(false);
    }
  }

  function handleLogout() {
    localStorage.removeItem(TOKEN_KEY);
    setToken("");
    setSnapshot(null);
    setError("");
  }

  useEffect(() => {
    if (token) {
      loadDashboard(token);
    }
  }, [token]);

  if (!token) {
    return <LoginView onLogin={handleLogin} loading={loading} error={error} />;
  }

  return (
    <div className="app-shell">
      <header className="hero">
        <div>
          <p className="eyebrow">Harbor Dashboard</p>
          <h1>Panorama de Deploy em Tempo Real</h1>
          <p className="subtitle">
            Visao consolidada de devices, deployments e trilha de auditoria.
          </p>
        </div>

        <div className="hero__actions">
          <button onClick={() => loadDashboard()} disabled={loading}>
            {loading ? "Atualizando..." : "Atualizar"}
          </button>
          <button className="ghost" onClick={handleLogout}>
            Sair
          </button>
        </div>
      </header>

      {error ? <p className="form-error form-error--inline">{error}</p> : null}

      <section className="stats-grid">
        <StatCard
          label="Devices Totais"
          value={stats.devicesTotal}
          hint={`${stats.devicesPending} pendentes`}
        />
        <StatCard
          label="Devices Aceitos"
          value={stats.devicesAccepted}
          hint={`${stats.devicesRejected} rejeitados`}
        />
        <StatCard
          label="Deploys Totais"
          value={stats.deployTotal}
          hint={`${stats.deployActive} ativos`}
        />
        <StatCard
          label="Deploys Concluidos"
          value={stats.deployCompleted}
          hint={`${stats.deployCancelled} cancelados`}
        />
      </section>

      <section className="panel-grid">
        <article className="panel">
          <h2>Ultimos Deployments</h2>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Nome</th>
                  <th>Status</th>
                  <th>Criado em</th>
                </tr>
              </thead>
              <tbody>
                {(snapshot?.latestDeployments || []).map((deployment) => (
                  <tr key={deployment.id}>
                    <td>{deployment.name}</td>
                    <td>
                      <span className={statusTone(deployment.status)}>{deployment.status}</span>
                    </td>
                    <td>{formatDate(deployment.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </article>

        <article className="panel">
          <h2>Ultimos Devices</h2>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Tipo</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {(snapshot?.latestDevices || []).map((device) => (
                  <tr key={device.id}>
                    <td className="mono">{device.id.slice(0, 8)}...</td>
                    <td>{device.device_type || "-"}</td>
                    <td>
                      <span className={statusTone(device.status)}>{device.status}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </article>
      </section>

      <section className="panel-grid panel-grid--single">
        <article className="panel">
          <h2>Auditoria Recente</h2>
          <ul className="timeline">
            {(snapshot?.latestAudit || []).map((entry) => (
              <li key={entry.id}>
                <p>
                  <strong>{entry.actor}</strong> executou <strong>{entry.action}</strong>
                </p>
                <p className="muted">
                  recurso: {entry.resource} | id: {entry.resource_id} | {formatDate(entry.created_at)}
                </p>
              </li>
            ))}
          </ul>
        </article>
      </section>
    </div>
  );
}
