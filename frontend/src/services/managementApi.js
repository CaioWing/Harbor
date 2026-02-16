import { request } from "./httpClient";

function toQuery(params = {}) {
  const search = new URLSearchParams();

  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") {
      return;
    }

    if (Array.isArray(value)) {
      value.forEach((entry) => {
        if (entry !== undefined && entry !== null && entry !== "") {
          search.append(key, String(entry));
        }
      });
      return;
    }

    search.append(key, String(value));
  });

  const query = search.toString();
  return query ? `?${query}` : "";
}

export function loginManagement(credentials) {
  return request("/management/auth/login", {
    method: "POST",
    body: credentials
  });
}

export function refreshManagementToken(token) {
  return request("/management/auth/refresh", {
    method: "POST",
    token
  });
}

export function getDeviceCounts(token) {
  return request("/management/devices/count", { token });
}

export function listDevices(token, params) {
  return request(`/management/devices${toQuery(params)}`, { token });
}

export function getDeviceById(token, id) {
  return request(`/management/devices/${id}`, { token });
}

export function updateDeviceStatus(token, id, status) {
  return request(`/management/devices/${id}/status`, {
    method: "PUT",
    token,
    body: { status }
  });
}

export function listDeployments(token, params) {
  return request(`/management/deployments${toQuery(params)}`, { token });
}

export function getDeploymentById(token, id) {
  return request(`/management/deployments/${id}`, { token });
}

export function getDeploymentDevices(token, id) {
  return request(`/management/deployments/${id}/devices`, { token });
}

export function getDeploymentStats(token) {
  return request("/management/deployments/statistics", { token });
}

export function cancelDeployment(token, id) {
  return request(`/management/deployments/${id}/cancel`, {
    method: "POST",
    token
  });
}

export function listAudit(token, params) {
  return request(`/management/audit${toQuery(params)}`, { token });
}
