import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import AppLayout from "../components/layout/AppLayout";
import StatusPill from "../components/ui/StatusPill";
import BarListChart from "../components/charts/BarListChart";
import { useAuth } from "../context/AuthContext";
import { useAutoRefresh } from "../hooks/useAutoRefresh";
import { cancelDeployment, getDeploymentById, getDeploymentDevices } from "../services/managementApi";
import { formatDateTime, shortenID } from "../utils/format";

function canCancel(status) {
  return status === "scheduled" || status === "active";
}

export default function DeploymentDetailsPage() {
  const { id } = useParams();
  const { token, signOut } = useAuth();
  const [deployment, setDeployment] = useState(null);
  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [refreshMs, setRefreshMs] = useState(30000);
  const [actionLoading, setActionLoading] = useState(false);
  const [actionError, setActionError] = useState("");
  const [actionSuccess, setActionSuccess] = useState("");

  const load = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      }
      setError("");

      try {
        const [deploymentResponse, devicesResponse] = await Promise.all([
          getDeploymentById(token, id),
          getDeploymentDevices(token, id)
        ]);

        setDeployment(deploymentResponse || null);
        setDevices(devicesResponse?.data || []);
      } catch (loadError) {
        setError(loadError.message || "Falha ao carregar deployment");
        if (loadError.status === 401) {
          signOut();
        }
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [token, id, signOut]
  );

  useEffect(() => {
    load();
  }, [load]);

  useAutoRefresh(() => load(true), refreshMs, refreshMs > 0);

  const statusData = useMemo(() => {
    const counter = devices.reduce((acc, item) => {
      const key = item.status || "unknown";
      acc[key] = (acc[key] || 0) + 1;
      return acc;
    }, {});

    return Object.entries(counter).map(([label, value]) => ({ label, value }));
  }, [devices]);

  async function handleCancel() {
    setActionError("");
    setActionSuccess("");

    const confirmed = globalThis.window?.confirm("Cancelar este deployment?");
    if (!confirmed) {
      return;
    }

    setActionLoading(true);

    try {
      await cancelDeployment(token, id);
      setActionSuccess("Deployment cancelado com sucesso.");
      await load();
    } catch (actionErr) {
      setActionError(actionErr.message || "Falha ao cancelar deployment");
      if (actionErr.status === 401) {
        signOut();
      }
    } finally {
      setActionLoading(false);
    }
  }

  return (
    <AppLayout
      title="Detalhe do Deployment"
      subtitle={id}
      actions={
        <>
          <label className="control-inline">
            Auto-refresh
            <select value={refreshMs} onChange={(event) => setRefreshMs(Number(event.target.value))}>
              <option value={0}>Off</option>
              <option value={15000}>15s</option>
              <option value={30000}>30s</option>
              <option value={60000}>60s</option>
            </select>
          </label>
          <button onClick={() => load()} disabled={loading}>
            {loading ? "Atualizando..." : "Atualizar"}
          </button>
          {canCancel(deployment?.status) ? (
            <button type="button" className="danger" onClick={handleCancel} disabled={actionLoading}>
              {actionLoading ? "Cancelando..." : "Cancelar"}
            </button>
          ) : null}
          <Link to="/deployments" className="button-link">
            Voltar
          </Link>
        </>
      }
    >
      {error ? <p className="form-error form-error--inline">{error}</p> : null}
      {actionError ? <p className="form-error form-error--inline">{actionError}</p> : null}
      {actionSuccess ? <p className="form-success form-success--inline">{actionSuccess}</p> : null}

      {deployment ? (
        <section className="stats-grid stats-grid--three">
          <article className="stat-card">
            <p className="stat-card__label">Status</p>
            <p>
              <StatusPill status={deployment.status} />
            </p>
            <p className="stat-card__hint">Artifact: {shortenID(deployment.artifact_id, 12)}</p>
          </article>

          <article className="stat-card">
            <p className="stat-card__label">Criado</p>
            <p className="stat-card__value small">{formatDateTime(deployment.created_at)}</p>
            <p className="stat-card__hint">Iniciado: {formatDateTime(deployment.started_at)}</p>
          </article>

          <article className="stat-card">
            <p className="stat-card__label">Finalizado</p>
            <p className="stat-card__value small">{formatDateTime(deployment.finished_at)}</p>
            <p className="stat-card__hint">Max paralelo: {deployment.max_parallel ?? 0}</p>
          </article>
        </section>
      ) : null}

      <section className="panel-grid">
        <BarListChart title="Distribuicao por Status" data={statusData} />

        <article className="panel">
          <h3>Escopo de Target</h3>
          <ul className="key-list">
            <li>
              <span>Device IDs</span>
              <strong>{(deployment?.target_device_ids || []).length}</strong>
            </li>
            <li>
              <span>Device Tags</span>
              <strong>{(deployment?.target_device_tags || []).join(", ") || "-"}</strong>
            </li>
            <li>
              <span>Device Types</span>
              <strong>{(deployment?.target_device_types || []).join(", ") || "-"}</strong>
            </li>
          </ul>
        </article>
      </section>

      <article className="panel">
        <div className="panel__header">
          <h3>Devices do Deployment</h3>
          <span className="muted">{devices.length} registros</span>
        </div>

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>DD ID</th>
                <th>Device ID</th>
                <th>Status</th>
                <th>Tentativas</th>
                <th>Log</th>
              </tr>
            </thead>
            <tbody>
              {devices.map((entry) => (
                <tr key={entry.id}>
                  <td className="mono">{shortenID(entry.id)}</td>
                  <td>
                    <Link to={`/devices/${entry.device_id}`} className="mono">
                      {shortenID(entry.device_id)}
                    </Link>
                  </td>
                  <td>
                    <StatusPill status={entry.status} />
                  </td>
                  <td>{entry.attempts}</td>
                  <td>{entry.log || "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </article>
    </AppLayout>
  );
}
