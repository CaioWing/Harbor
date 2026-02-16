const API_BASE = import.meta.env.VITE_API_BASE_URL || "/api/v1";

async function request(path, options = {}) {
  const { token, method = "GET", body, headers = {} } = options;

  const requestHeaders = { ...headers };
  if (body !== undefined) {
    requestHeaders["Content-Type"] = "application/json";
  }

  if (token) {
    requestHeaders.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers: requestHeaders,
    body: body !== undefined ? JSON.stringify(body) : undefined
  });

  const contentType = response.headers.get("content-type") || "";
  const hasJSON = contentType.includes("application/json");
  const payload = hasJSON ? await response.json() : null;

  if (!response.ok) {
    const message = payload?.error || `Request failed (${response.status})`;
    throw new Error(message);
  }

  return payload;
}

export async function loginManagement({ email, password }) {
  return request("/management/auth/login", {
    method: "POST",
    body: { email, password }
  });
}

export async function fetchDashboardSnapshot(token) {
  const [deviceCounts, deploymentStats, deployments, devices, audit] = await Promise.all([
    request("/management/devices/count", { token }),
    request("/management/deployments/statistics", { token }),
    request("/management/deployments?per_page=5&order=desc", { token }),
    request("/management/devices?per_page=5&order=desc", { token }),
    request("/management/audit?per_page=5&order=desc", { token })
  ]);

  return {
    deviceCounts,
    deploymentStats,
    latestDeployments: deployments?.data || [],
    latestDevices: devices?.data || [],
    latestAudit: audit?.data || []
  };
}
