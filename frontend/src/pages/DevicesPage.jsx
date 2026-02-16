import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import AppLayout from "../components/layout/AppLayout";
import StatusPill from "../components/ui/StatusPill";
import { useAuth } from "../context/AuthContext";
import { listDevices } from "../services/managementApi";
import { formatDateTime, shortenID } from "../utils/format";

const DEFAULT_FILTERS = {
  status: "",
  device_type: "",
  tag: ""
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
    status: searchParams.get("status") || "",
    device_type: searchParams.get("device_type") || "",
    tag: searchParams.get("tag") || ""
  };
}

export default function DevicesPage() {
  const { token, signOut } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();

  const filters = useMemo(() => parseFilters(searchParams), [searchParams]);
  const page = useMemo(() => parsePage(searchParams.get("page")), [searchParams]);

  const [draft, setDraft] = useState(() => ({ ...DEFAULT_FILTERS }));
  const [reloadTick, setReloadTick] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
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
  }, [filters.status, filters.device_type, filters.tag]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");

      try {
        const response = await listDevices(token, {
          page,
          per_page: 20,
          order: "desc",
          status: filters.status,
          device_type: filters.device_type,
          tag: filters.tag
            ? filters.tag
                .split(",")
                .map((entry) => entry.trim())
                .filter(Boolean)
            : []
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

        setError(loadError.message || "Falha ao carregar devices");
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
  }, [token, signOut, page, filters.status, filters.device_type, filters.tag, reloadTick]);

  useEffect(() => {
    const totalPages = Math.max(1, Number(pagination.total_pages || 1));
    if (page > totalPages) {
      setQueryState({ page: totalPages > 1 ? totalPages : "" }, { replace: true });
    }
  }, [page, pagination.total_pages, setQueryState]);

  const filterSummary = useMemo(() => {
    const active = [];
    if (filters.status) active.push(`status=${filters.status}`);
    if (filters.device_type) active.push(`device_type=${filters.device_type}`);
    if (filters.tag) active.push(`tags=${filters.tag}`);
    return active.length ? active.join(" | ") : "Sem filtros ativos";
  }, [filters]);

  function applyFilters(event) {
    event.preventDefault();
    setQueryState({
      status: draft.status,
      device_type: draft.device_type,
      tag: draft.tag,
      page: ""
    });
  }

  function clearFilters() {
    setDraft({ ...DEFAULT_FILTERS });
    setQueryState({
      status: "",
      device_type: "",
      tag: "",
      page: ""
    });
  }

  function goToPage(nextPage) {
    setQueryState({ page: nextPage > 1 ? nextPage : "" });
  }

  return (
    <AppLayout
      title="Devices"
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
            Status
            <select
              value={draft.status}
              onChange={(event) => setDraft((state) => ({ ...state, status: event.target.value }))}
            >
              <option value="">Todos</option>
              <option value="pending">pending</option>
              <option value="accepted">accepted</option>
              <option value="rejected">rejected</option>
              <option value="decommissioned">decommissioned</option>
            </select>
          </label>

          <label>
            Device Type
            <input
              value={draft.device_type}
              onChange={(event) => setDraft((state) => ({ ...state, device_type: event.target.value }))}
              placeholder="raspberry-pi-4"
            />
          </label>

          <label>
            Tags (CSV)
            <input
              value={draft.tag}
              onChange={(event) => setDraft((state) => ({ ...state, tag: event.target.value }))}
              placeholder="production, rack-01"
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
          <h3>Lista de Devices</h3>
          <span className="muted">{pagination.total} registros</span>
        </div>

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Tipo</th>
                <th>Status</th>
                <th>Tags</th>
                <th>Ultimo Check-in</th>
              </tr>
            </thead>
            <tbody>
              {items.map((device) => (
                <tr key={device.id}>
                  <td>
                    <Link to={`/devices/${device.id}`} className="mono">
                      {shortenID(device.id)}
                    </Link>
                  </td>
                  <td>{device.device_type || "-"}</td>
                  <td>
                    <StatusPill status={device.status} />
                  </td>
                  <td>{(device.tags || []).join(", ") || "-"}</td>
                  <td>{formatDateTime(device.last_check_in)}</td>
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
