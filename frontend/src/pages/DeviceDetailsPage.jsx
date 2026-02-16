import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import AppLayout from "../components/layout/AppLayout";
import StatusPill from "../components/ui/StatusPill";
import { useAuth } from "../context/AuthContext";
import { getDeviceById, updateDeviceStatus } from "../services/managementApi";
import { formatDateTime } from "../utils/format";

export default function DeviceDetailsPage() {
  const { id } = useParams();
  const { token, signOut } = useAuth();
  const [device, setDevice] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [actionLoading, setActionLoading] = useState("");
  const [actionError, setActionError] = useState("");
  const [actionSuccess, setActionSuccess] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const response = await getDeviceById(token, id);
      setDevice(response);
    } catch (loadError) {
      setError(loadError.message || "Falha ao carregar device");
      if (loadError.status === 401) {
        signOut();
      }
    } finally {
      setLoading(false);
    }
  }, [token, id, signOut]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleStatusUpdate(nextStatus) {
    setActionError("");
    setActionSuccess("");
    setActionLoading(nextStatus);

    try {
      await updateDeviceStatus(token, id, nextStatus);
      setActionSuccess(`Status atualizado para ${nextStatus}.`);
      await load();
    } catch (actionErr) {
      setActionError(actionErr.message || "Falha ao atualizar status");
      if (actionErr.status === 401) {
        signOut();
      }
    } finally {
      setActionLoading("");
    }
  }

  return (
    <AppLayout
      title="Detalhe do Device"
      subtitle={id}
      actions={
        <div className="row-actions row-actions--header">
          <Link to="/devices" className="button-link">
            Voltar para lista
          </Link>
          <button
            type="button"
            onClick={() => handleStatusUpdate("accepted")}
            disabled={!device || loading || actionLoading === "accepted" || device.status === "accepted"}
          >
            {actionLoading === "accepted" ? "Salvando..." : "Aceitar"}
          </button>
          <button
            type="button"
            className="danger"
            onClick={() => handleStatusUpdate("rejected")}
            disabled={!device || loading || actionLoading === "rejected" || device.status === "rejected"}
          >
            {actionLoading === "rejected" ? "Salvando..." : "Rejeitar"}
          </button>
        </div>
      }
    >
      {loading ? <p className="muted">Carregando...</p> : null}
      {error ? <p className="form-error form-error--inline">{error}</p> : null}
      {actionError ? <p className="form-error form-error--inline">{actionError}</p> : null}
      {actionSuccess ? <p className="form-success form-success--inline">{actionSuccess}</p> : null}

      {device ? (
        <>
          <section className="stats-grid stats-grid--three">
            <article className="stat-card">
              <p className="stat-card__label">Status</p>
              <p>
                <StatusPill status={device.status} />
              </p>
              <p className="stat-card__hint">Tipo: {device.device_type || "-"}</p>
            </article>

            <article className="stat-card">
              <p className="stat-card__label">Criado em</p>
              <p className="stat-card__value small">{formatDateTime(device.created_at)}</p>
              <p className="stat-card__hint">Atualizado: {formatDateTime(device.updated_at)}</p>
            </article>

            <article className="stat-card">
              <p className="stat-card__label">Ultimo Check-in</p>
              <p className="stat-card__value small">{formatDateTime(device.last_check_in)}</p>
              <p className="stat-card__hint">Tags: {(device.tags || []).join(", ") || "-"}</p>
            </article>
          </section>

          <section className="panel-grid">
            <article className="panel">
              <h3>Identity Data</h3>
              <pre className="json-block">{JSON.stringify(device.identity_data || {}, null, 2)}</pre>
            </article>
            <article className="panel">
              <h3>Inventory</h3>
              <pre className="json-block">{JSON.stringify(device.inventory || {}, null, 2)}</pre>
            </article>
          </section>
        </>
      ) : null}
    </AppLayout>
  );
}
