import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import AppLayout from "../components/layout/AppLayout";
import { useAuth } from "../context/AuthContext";
import { listAudit } from "../services/managementApi";
import { formatDateTime, shortenID } from "../utils/format";

const DEFAULT_FILTERS = {
  actor: "",
  action: "",
  resource: ""
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
    actor: searchParams.get("actor") || "",
    action: searchParams.get("action") || "",
    resource: searchParams.get("resource") || ""
  };
}

export default function AuditPage() {
  const { token, signOut } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();

  const filters = useMemo(() => parseFilters(searchParams), [searchParams]);
  const page = useMemo(() => parsePage(searchParams.get("page")), [searchParams]);

  const [draft, setDraft] = useState(() => ({ ...DEFAULT_FILTERS }));
  const [reloadTick, setReloadTick] = useState(0);
  const [items, setItems] = useState([]);
  const [pagination, setPagination] = useState({ page: 1, per_page: 20, total: 0, total_pages: 1 });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

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
  }, [filters.actor, filters.action, filters.resource]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");

      try {
        const response = await listAudit(token, {
          page,
          per_page: 20,
          order: "desc",
          actor: filters.actor,
          action: filters.action,
          resource: filters.resource
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

        setError(loadError.message || "Falha ao carregar auditoria");
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
  }, [token, signOut, page, filters.actor, filters.action, filters.resource, reloadTick]);

  useEffect(() => {
    const totalPages = Math.max(1, Number(pagination.total_pages || 1));
    if (page > totalPages) {
      setQueryState({ page: totalPages > 1 ? totalPages : "" }, { replace: true });
    }
  }, [page, pagination.total_pages, setQueryState]);

  const filterSummary = useMemo(() => {
    const active = [];
    if (filters.actor) active.push(`actor=${filters.actor}`);
    if (filters.action) active.push(`action=${filters.action}`);
    if (filters.resource) active.push(`resource=${filters.resource}`);
    return active.length ? active.join(" | ") : "Sem filtros ativos";
  }, [filters]);

  function applyFilters(event) {
    event.preventDefault();
    setQueryState({
      actor: draft.actor,
      action: draft.action,
      resource: draft.resource,
      page: ""
    });
  }

  function clearFilters() {
    setDraft({ ...DEFAULT_FILTERS });
    setQueryState({
      actor: "",
      action: "",
      resource: "",
      page: ""
    });
  }

  function goToPage(nextPage) {
    setQueryState({ page: nextPage > 1 ? nextPage : "" });
  }

  return (
    <AppLayout
      title="Audit Log"
      subtitle={filterSummary}
      actions={
        <button onClick={() => setReloadTick((value) => value + 1)} disabled={loading}>
          Recarregar
        </button>
      }
    >
      <article className="panel">
        <h3>Filtros</h3>
        <form className="filters" onSubmit={applyFilters}>
          <label>
            Actor
            <input
              value={draft.actor}
              onChange={(event) => setDraft((state) => ({ ...state, actor: event.target.value }))}
              placeholder="admin"
            />
          </label>

          <label>
            Action
            <input
              value={draft.action}
              onChange={(event) => setDraft((state) => ({ ...state, action: event.target.value }))}
              placeholder="deployment.create"
            />
          </label>

          <label>
            Resource
            <input
              value={draft.resource}
              onChange={(event) => setDraft((state) => ({ ...state, resource: event.target.value }))}
              placeholder="deployment"
            />
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

      <article className="panel">
        <div className="panel__header">
          <h3>Eventos</h3>
          <span className="muted">{pagination.total} registros</span>
        </div>

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Actor</th>
                <th>Action</th>
                <th>Resource</th>
                <th>Resource ID</th>
                <th>Data</th>
              </tr>
            </thead>
            <tbody>
              {items.map((entry) => (
                <tr key={entry.id}>
                  <td>{entry.actor}</td>
                  <td>{entry.action}</td>
                  <td>{entry.resource}</td>
                  <td className="mono">{shortenID(entry.resource_id, 12)}</td>
                  <td>{formatDateTime(entry.created_at)}</td>
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
