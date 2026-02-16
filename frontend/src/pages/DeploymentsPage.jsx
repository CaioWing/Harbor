import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import AppLayout from "../components/layout/AppLayout";
import StatusPill from "../components/ui/StatusPill";
import { useAuth } from "../context/AuthContext";
import { cancelDeployment, listDeployments } from "../services/managementApi";
import { formatDateTime, shortenID } from "../utils/format";

const DEFAULT_FILTERS = {
  status: ""
};

function parsePage(rawValue) {
  const value = Number(rawValue || "1");
  if (!Number.isFinite(value) || value < 1) {
    return 1;
  }
  return Math.floor(value);
}

function parseFilters(searchParams) {
  return {
    status: searchParams.get("status") || ""
  };
}

function canCancel(status) {
  return status === "scheduled" || status === "active";
}

export default function DeploymentsPage() {
  const { token, signOut } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();

  const filters = useMemo(() => parseFilters(searchParams), [searchParams]);
  const page = useMemo(() => parsePage(searchParams.get("page")), [searchParams]);

  const [draft, setDraft] = useState(() => ({ ...DEFAULT_FILTERS }));
  const [reloadTick, setReloadTick] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [actionError, setActionError] = useState("");
  const [actionSuccess, setActionSuccess] = useState("");
  const [actionLoadingID, setActionLoadingID] = useState("");
  const [items, setItems] = useState([]);
  const [pagination, setPagination] = useState({ page: 1, per_page: 20, total: 0, total_pages: 1 });

  const setQueryState = useCallback(
    (patch, options = {}) => {
      const next = new URLSearchParams(searchParams);

      Object.entries(patch).forEach(([key, value]) => {
        if (value === undefined || value === null || value === "") {
          next.delete(key);
          return;
        }

        next.set(key, String(value));
      });

      setSearchParams(next, options);
    },
    [searchParams, setSearchParams]
  );

  useEffect(() => {
    setDraft({ ...filters });
  }, [filters.status]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");

      try {
        const response = await listDeployments(token, {
          page,
          per_page: 20,
          order: "desc",
          status: filters.status
        });

        if (cancelled) {
          return;
        }

        setItems(response?.data || []);
        setPagination(response?.pagination || { page: 1, per_page: 20, total: 0, total_pages: 1 });
      } catch (loadError) {
        if (cancelled) {
          return;
        }

        setError(loadError.message || "Falha ao carregar deployments");
        if (loadError.status === 401) {
          signOut();
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    load();

    return () => {
      cancelled = true;
    };
  }, [token, signOut, page, filters.status, reloadTick]);

  useEffect(() => {
    const totalPages = Math.max(1, Number(pagination.total_pages || 1));
    if (page > totalPages) {
      setQueryState({ page: totalPages > 1 ? totalPages : "" }, { replace: true });
    }
  }, [page, pagination.total_pages, setQueryState]);

  const filterSummary = useMemo(() => {
    return filters.status ? `status=${filters.status}` : "Sem filtros ativos";
  }, [filters.status]);

  function applyFilters(event) {
    event.preventDefault();
    setQueryState({ status: draft.status, page: "" });
  }

  function clearFilters() {
    setDraft({ ...DEFAULT_FILTERS });
    setQueryState({ status: "", page: "" });
  }

  function goToPage(nextPage) {
    setQueryState({ page: nextPage > 1 ? nextPage : "" });
  }

  async function handleCancel(id) {
    setActionError("");
    setActionSuccess("");

    const confirmed = globalThis.window?.confirm("Cancelar este deployment?");
    if (!confirmed) {
      return;
    }

    setActionLoadingID(id);

    try {
      await cancelDeployment(token, id);
      setActionSuccess(`Deployment ${shortenID(id)} cancelado com sucesso.`);
      setReloadTick((value) => value + 1);
    } catch (actionErr) {
      setActionError(actionErr.message || "Falha ao cancelar deployment");
      if (actionErr.status === 401) {
        signOut();
      }
    } finally {
      setActionLoadingID("");
    }
  }

  return (
    <AppLayout
      title="Deployments"
      subtitle={filterSummary}
      actions={
        <button onClick={() => setReloadTick((value) => value + 1)} disabled={loading}>
          Recarregar
        </button>
      }
    >
      <article className="panel">
        <h3>Filtros</h3>
        <form className="filters filters--two" onSubmit={applyFilters}>
          <label>
            Status
            <select
              value={draft.status}
              onChange={(event) => setDraft((state) => ({ ...state, status: event.target.value }))}
            >
              <option value="">Todos</option>
              <option value="scheduled">scheduled</option>
              <option value="active">active</option>
              <option value="completed">completed</option>
              <option value="cancelled">cancelled</option>
            </select>
          </label>

          <div className="filters__actions">
            <button type="submit" disabled={loading}>
              Aplicar
            </button>
            <button type="button" className="ghost" onClick={clearFilters} disabled={loading}>
              Limpar
            </button>
          </div>
        </form>
      </article>

      {error ? <p className="form-error form-error--inline">{error}</p> : null}
      {actionError ? <p className="form-error form-error--inline">{actionError}</p> : null}
      {actionSuccess ? <p className="form-success form-success--inline">{actionSuccess}</p> : null}

      <article className="panel">
        <div className="panel__header">
          <h3>Lista de Deployments</h3>
          <span className="muted">{pagination.total} registros</span>
        </div>

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Nome</th>
                <th>Status</th>
                <th>Artifact</th>
                <th>Criado em</th>
                <th>Acoes</th>
              </tr>
            </thead>
            <tbody>
              {items.map((deployment) => (
                <tr key={deployment.id}>
                  <td className="mono">{shortenID(deployment.id)}</td>
                  <td>
                    <Link to={`/deployments/${deployment.id}`}>{deployment.name}</Link>
                  </td>
                  <td>
                    <StatusPill status={deployment.status} />
                  </td>
                  <td className="mono">{shortenID(deployment.artifact_id, 10)}</td>
                  <td>{formatDateTime(deployment.created_at)}</td>
                  <td>
                    <div className="row-actions">
                      <Link className="button-link" to={`/deployments/${deployment.id}`}>
                        Detalhes
                      </Link>
                      {canCancel(deployment.status) ? (
                        <button
                          type="button"
                          className="danger"
                          onClick={() => handleCancel(deployment.id)}
                          disabled={actionLoadingID === deployment.id}
                        >
                          {actionLoadingID === deployment.id ? "Cancelando..." : "Cancelar"}
                        </button>
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <footer className="pagination">
          <button onClick={() => goToPage(Math.max(1, page - 1))} disabled={page <= 1 || loading}>
            Anterior
          </button>
          <span>
            Pagina {page} de {Math.max(1, pagination.total_pages)}
          </span>
          <button
            onClick={() => goToPage(page + 1)}
            disabled={page >= Math.max(1, pagination.total_pages) || loading}
          >
            Proxima
          </button>
        </footer>
      </article>
    </AppLayout>
  );
}
