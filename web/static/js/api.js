let _refreshPromise = null;

function getAuthToken() {
  try {
    return localStorage.getItem("access_token");
  } catch {
    return null;
  }
}

function getRefreshToken() {
  try {
    return localStorage.getItem("refresh_token");
  } catch {
    return null;
  }
}

async function refreshAccessToken() {
  if (_refreshPromise) return _refreshPromise;

  const rt = getRefreshToken();
  if (!rt) throw new Error("No refresh token");

  _refreshPromise = (async () => {
    const res = await fetch("/api/v1/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ refresh_token: rt }),
    });
    if (!res.ok) throw new Error("Refresh failed");
    const data = await res.json();
    if (!data.access_token) throw new Error("Refresh response missing access_token");
    localStorage.setItem("access_token", data.access_token);
    if (data.refresh_token) localStorage.setItem("refresh_token", data.refresh_token);
    return data.access_token;
  })();

  _refreshPromise = _refreshPromise.then(
    (token) => {
      _refreshPromise = null;
      return token;
    },
    (e) => {
      _refreshPromise = null;
      throw e;
    }
  );

  return _refreshPromise;
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

  let res = await fetch(url, {
    ...requestInit,
    method: rawMethod || "GET",
    headers,
    body,
  });

  if (res.status === 401 && !path.includes("/auth/refresh") && !path.includes("/auth/login")) {
    try {
      const newToken = await refreshAccessToken();
      headers.set("Authorization", `Bearer ${newToken}`);
      res = await fetch(url, {
        ...requestInit,
        method: rawMethod || "GET",
        headers,
        body,
      });
    } catch {
      try {
        Alpine.store("auth").logout();
      } catch {
        // Best-effort: ignore if Alpine store isn't available
      }
    }
  }

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
