function getAuthToken() {
  try {
    return localStorage.getItem("access_token");
  } catch {
    return null;
  }
}

async function apiFetch(path, options = {}) {
  const { headers: rawHeaders, body: rawBody, method: rawMethod, ...requestInit } = options || {};
  const url = path.startsWith("/api/") ? path : `/api/v1${path.startsWith("/") ? "" : "/"}${path}`;

  const headers = new Headers(rawHeaders || {});
  headers.set("Accept", "application/json");

  const token = getAuthToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const hasContentType = headers.has("Content-Type");
  const shouldJsonify =
    rawBody != null &&
    !(rawBody instanceof FormData) &&
    (Array.isArray(rawBody) || Object.prototype.toString.call(rawBody) === "[object Object]");

  const body = shouldJsonify ? JSON.stringify(rawBody) : rawBody;
  if (shouldJsonify && !hasContentType) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(url, {
    ...requestInit,
    method: rawMethod || "GET",
    headers,
    body,
  });

  const contentType = res.headers.get("content-type") || "";
  const isJson = contentType.includes("application/json");

  if (res.status === 204) {
    return { ok: true, status: res.status, data: null };
  }

  const payload = isJson ? await res.json().catch(() => null) : await res.text().catch(() => null);

  if (!res.ok) {
    const message =
      (payload && payload.message) ||
      (payload && payload.error) ||
      (typeof payload === "string" ? payload : null) ||
      `HTTP ${res.status}`;

    const err = new Error(message);
    err.status = res.status;
    err.payload = payload;
    throw err;
  }

  return { ok: true, status: res.status, data: payload };
}

window.apiFetch = apiFetch;

