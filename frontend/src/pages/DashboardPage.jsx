import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import AppLayout from "../components/layout/AppLayout";
import StatCard from "../components/ui/StatCard";
import StatusPill from "../components/ui/StatusPill";
import DonutChart from "../components/charts/DonutChart";
import BarListChart from "../components/charts/BarListChart";
import { useAuth } from "../context/AuthContext";
import { useAutoRefresh } from "../hooks/useAutoRefresh";
import {
  getDeploymentStats,
  getDeviceCounts,
  listAudit,
  listDeployments,
  listDevices
} from "../services/managementApi";
import { formatDateTime, shortenID } from "../utils/format";

const REFRESH_OPTIONS = [
  { label: "Off", value: 0 },
  { label: "15s", value: 15000 },
  { label: "30s", value: 30000 },
  { label: "60s", value: 60000 }
];

export default function DashboardPage() {
  const { token, signOut } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [refreshMs, setRefreshMs] = useState(15000);
  const [lastUpdatedAt, setLastUpdatedAt] = useState("");
  const [snapshot, setSnapshot] = useState({
    deviceCounts: {},
    deploymentStats: {},
    latestDeployments: [],
    latestDevices: [],
    latestAudit: []
  });

  const loadSnapshot = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      }
      setError("");

      try {
        const [deviceCounts, deploymentStats, deployments, devices, audit] = await Promise.all([
          getDeviceCounts(token),
          getDeploymentStats(token),
          listDeployments(token, { page: 1, per_page: 5, order: "desc" }),
          listDevices(token, { page: 1, per_page: 5, order: "desc" }),
          listAudit(token, { page: 1, per_page: 5, order: "desc" })
        ]);

        setSnapshot({
          deviceCounts,
          deploymentStats,
          latestDeployments: deployments?.data || [],
          latestDevices: devices?.data || [],
          latestAudit: audit?.data || []
        });
        setLastUpdatedAt(new Date().toISOString());
      } catch (loadError) {
        setError(loadError.message || "Falha ao carregar dashboard");
        if (loadError.status === 401) {
          signOut();
        }
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [token, signOut]
  );

  useEffect(() => {
    loadSnapshot();
  }, [loadSnapshot]);

  useAutoRefresh(() => loadSnapshot(true), refreshMs, refreshMs > 0);

  const cards = useMemo(() => {
    const deviceCounts = snapshot.deviceCounts || {};
    const deploy = snapshot.deploymentStats || {};

    const devicesTotal =
      Number(deviceCounts.pending || 0) +
      Number(deviceCounts.accepted || 0) +
      Number(deviceCounts.rejected || 0) +
      Number(deviceCounts.decommissioned || 0);

    return [
      {
        label: "Devices Totais",
        value: devicesTotal,
        hint: `${Number(deviceCounts.pending || 0)} pendentes`
      },
      {
        label: "Devices Aceitos",
        value: Number(deviceCounts.accepted || 0),
        hint: `${Number(deviceCounts.rejected || 0)} rejeitados`
      },
      {
        label: "Deployments Totais",
        value: Number(deploy.total || 0),
        hint: `${Number(deploy.active || 0)} ativos`
      },
      {
        label: "Deployments Completos",
        value: Number(deploy.completed || 0),
        hint: `${Number(deploy.cancelled || 0)} cancelados`
      }
    ];
  }, [snapshot]);

  const donutData = useMemo(() => {
    const counts = snapshot.deviceCounts || {};
    return [
      { label: "accepted", value: Number(counts.accepted || 0) },
      { label: "pending", value: Number(counts.pending || 0) },
      { label: "rejected", value: Number(counts.rejected || 0) },
      { label: "decommissioned", value: Number(counts.decommissioned || 0) }
    ];
  }, [snapshot]);

  const deployBars = useMemo(() => {
    const stats = snapshot.deploymentStats || {};
    return [
      { label: "scheduled", value: Number(stats.scheduled || 0) },
      { label: "active", value: Number(stats.active || 0) },
      { label: "completed", value: Number(stats.completed || 0) },
      { label: "cancelled", value: Number(stats.cancelled || 0) }
    ];
  }, [snapshot]);

  return (
    <AppLayout
      title="Panorama em Tempo Real"
      subtitle={`Ultima atualizacao: ${formatDateTime(lastUpdatedAt)}`}
      actions={
        <>
          <label className="control-inline">
            Auto-refresh
            <select value={refreshMs} onChange={(event) => setRefreshMs(Number(event.target.value))}>
              {REFRESH_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
          <button onClick={() => loadSnapshot()} disabled={loading}>
            {loading ? "Atualizando..." : "Atualizar"}
          </button>
        </>
      }
    >
      {error ? <p className="form-error form-error--inline">{error}</p> : null}

      <section className="stats-grid">
        {cards.map((card) => (
          <StatCard key={card.label} label={card.label} value={card.value} hint={card.hint} />
        ))}
      </section>

      <section className="panel-grid">
        <DonutChart title="Distribuicao de Devices" data={donutData} />
        <BarListChart title="Status de Deployments" data={deployBars} />
      </section>

      <section className="panel-grid">
        <article className="panel">
          <div className="panel__header">
            <h3>Ultimos Deployments</h3>
            <Link to="/deployments" className="inline-link">
              Ver todos
            </Link>
          </div>
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
                {snapshot.latestDeployments.map((item) => (
                  <tr key={item.id}>
                    <td>
                      <Link to={`/deployments/${item.id}`}>{item.name}</Link>
                    </td>
                    <td>
                      <StatusPill status={item.status} />
                    </td>
                    <td>{formatDateTime(item.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </article>

        <article className="panel">
          <div className="panel__header">
            <h3>Ultimos Devices</h3>
            <Link to="/devices" className="inline-link">
              Ver todos
            </Link>
          </div>
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
                {snapshot.latestDevices.map((item) => (
                  <tr key={item.id}>
                    <td>
                      <Link to={`/devices/${item.id}`} className="mono">
                        {shortenID(item.id)}
                      </Link>
                    </td>
                    <td>{item.device_type || "-"}</td>
                    <td>
                      <StatusPill status={item.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </article>
      </section>

      <article className="panel">
        <div className="panel__header">
          <h3>Auditoria Recente</h3>
          <Link to="/audit" className="inline-link">
            Abrir auditoria
          </Link>
        </div>
        <ul className="timeline">
          {snapshot.latestAudit.map((entry) => (
            <li key={entry.id}>
              <p>
                <strong>{entry.actor}</strong> executou <strong>{entry.action}</strong>
              </p>
              <p className="muted">
                {entry.resource} | {shortenID(entry.resource_id, 12)} | {formatDateTime(entry.created_at)}
              </p>
            </li>
          ))}
        </ul>
      </article>
    </AppLayout>
  );
}
