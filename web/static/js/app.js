(function initToast() {
  const TOAST_ROOT_ID = "bit-toast-root";

  const typeClasses = {
    info: "border-sky-300 bg-sky-100 text-sky-900",
    success: "border-emerald-300 bg-emerald-100 text-emerald-900",
    error: "border-rose-300 bg-rose-100 text-rose-900",
    warn: "border-amber-300 bg-amber-100 text-amber-900",
  };

  const ensureRoot = () => {
    let root = document.getElementById(TOAST_ROOT_ID);
    if (root) return root;
    root = document.createElement("div");
    root.id = TOAST_ROOT_ID;
    root.className = "fixed bottom-4 left-4 z-[9999] flex flex-col gap-2";
    root.setAttribute("aria-live", "polite");
    root.setAttribute("role", "status");
    document.body.appendChild(root);
    return root;
  };

  const show = (message, { type = "info", durationMs = 4500 } = {}) => {
    const text = String(message || "").trim();
    if (!text) return;

    const root = ensureRoot();

    const el = document.createElement("div");
    el.className =
      "pointer-events-none select-none rounded-xl border px-4 py-3 text-sm shadow-lg transition-all duration-300 opacity-100 translate-y-0 " +
      (typeClasses[type] || typeClasses.info);
    el.textContent = text;

    root.appendChild(el);

    window.setTimeout(() => {
      try {
        el.classList.add("opacity-0", "translate-y-1");
      } catch {
        // Best-effort: animation failure shouldn't break toast lifecycle
      }
      window.setTimeout(() => {
        try {
          el.remove();
        } catch {
          // Best-effort: remove failure shouldn't crash UI
        }
      }, 320);
    }, durationMs);
  };

  window.toast = {
    show,
    info: (message, opts) => show(message, { ...(opts || {}), type: "info" }),
    success: (message, opts) => show(message, { ...(opts || {}), type: "success" }),
    error: (message, opts) => show(message, { ...(opts || {}), type: "error" }),
    warn: (message, opts) => show(message, { ...(opts || {}), type: "warn" }),
  };
})();

const toastApiError = (err, fallbackMessage) => {
  const msg = err && err.message ? String(err.message) : fallbackMessage;
  if (window.toast && typeof window.toast.error === "function" && msg) {
    window.toast.error(msg);
  }
  return msg;
};

const toastSuccess = (message) => {
  const msg = String(message || "").trim();
  if (!msg) return;
  if (window.toast && typeof window.toast.success === "function") {
    window.toast.success(msg);
  }
};

const STATUS_CLASSES = {
  "New": "bg-blue-100 text-blue-800 ring-1 ring-blue-300",
  "Open": "bg-yellow-100 text-yellow-800 ring-1 ring-yellow-300",
  "In Progress": "bg-indigo-100 text-indigo-800 ring-1 ring-indigo-300",
  "Resolved": "bg-emerald-100 text-emerald-800 ring-1 ring-emerald-300",
  "Closed": "bg-slate-200 text-slate-600 ring-1 ring-slate-400",
  "Reopened": "bg-red-100 text-red-800 ring-1 ring-red-300",
};

const PRIORITY_CLASSES = {
  "Trivial": "border border-slate-200 text-slate-400",
  "Minor": "border border-sky-300 text-sky-700",
  "Major": "border border-amber-300 text-amber-700",
  "Critical": "border border-orange-300 text-orange-700",
  "Blocker": "border border-red-400 text-red-700",
};

window.STATUS_CLASSES = STATUS_CLASSES;
window.PRIORITY_CLASSES = PRIORITY_CLASSES;

document.addEventListener("alpine:init", () => {
  Alpine.store("auth", {
    token: null,
    user: null,

    init() {
      this.token = this._getToken();
      this.user = this._getUser();
    },

    async register(email, password) {
      const { data } = await window.apiFetch("/auth/register", {
        method: "POST",
        body: { email, password },
      });
      return data;
    },

    async login(email, password) {
      const { data } = await window.apiFetch("/auth/login", {
        method: "POST",
        body: { email, password },
      });

      const token = data && data.access_token;
      if (!token) {
        throw new Error("Login response missing access_token");
      }

      this.token = token;
      this.user = data.user || null;
      this._setToken(token);
      this._setUser(this.user);
      if (data.refresh_token) {
        try { localStorage.setItem("refresh_token", data.refresh_token); } catch { }
      }
    },

    async logout() {
      try {
        const rt = (() => { try { return localStorage.getItem("refresh_token"); } catch { return null; } })();
        if (rt) {
          await window.apiFetch("/auth/logout", { method: "POST", body: { refresh_token: rt } });
        }
      } catch {
        // Best-effort — clear local state regardless
      }
      this.token = null;
      this.user = null;
      this._clearToken();
      this._clearUser();
      try { localStorage.removeItem("refresh_token"); } catch { }
      window.location.assign("/#/login");
    },

    _getToken() {
      try {
        return localStorage.getItem("access_token");
      } catch {
        return null;
      }
    },
    _setToken(token) {
      try {
        localStorage.setItem("access_token", token);
      } catch {
        console.debug("auth: localStorage.setItem(access_token) failed");
      }
    },
    _clearToken() {
      try {
        localStorage.removeItem("access_token");
      } catch {
        console.debug("auth: localStorage.removeItem(access_token) failed");
      }
    },

    _getUser() {
      try {
        const raw = localStorage.getItem("auth_user");
        if (!raw) return null;
        return JSON.parse(raw);
      } catch {
        return null;
      }
    },
    _setUser(user) {
      try {
        if (!user) {
          localStorage.removeItem("auth_user");
          return;
        }
        localStorage.setItem("auth_user", JSON.stringify(user));
      } catch {
        console.debug("auth: localStorage.setItem(auth_user) failed");
      }
    },
    _clearUser() {
      try {
        localStorage.removeItem("auth_user");
      } catch {
        console.debug("auth: localStorage.removeItem(auth_user) failed");
      }
    },
  });

  Alpine.store("auth").init();

  const RECENT_PROJECTS_KEY = "bit_recent_projects";
  const RECENT_PROJECTS_MAX = 5;
  const PUBLIC_PATHS = new Set(["/login", "/register", "/pending"]);

  Alpine.store("nav", {
    path: "/",
    recentProjects: [],

    init() {
      this.recentProjects = this._loadRecent();
      this.syncPath();
    },

    syncPath() {
      try {
        let h = String(window.location.hash || "");
        if (h.startsWith("#")) h = h.slice(1);
        if (!h) h = "/";
        if (!h.startsWith("/")) h = `/${h}`;
        this.path = h;
      } catch {
        this.path = "/";
      }
    },

    isPublicPath(path) {
      const p = String(path || this.path || "/").split("?")[0];
      return PUBLIC_PATHS.has(p);
    },

    showSidebar() {
      return Boolean(Alpine.store("auth").token) && !this.isPublicPath(this.path);
    },

    touchProject(project) {
      const id = String((project && project.id) || "").trim();
      if (!id) return;
      let name = String((project && project.name) || id).trim() || id;
      const existing = (this.recentProjects || []).find((p) => p && p.id === id);
      if (existing && existing.name && name.toLowerCase() === id.toLowerCase()) {
        if (existing.name.toLowerCase() !== id.toLowerCase()) {
          name = existing.name;
        }
      }
      const items = (this.recentProjects || []).filter((p) => p && p.id !== id);
      items.unshift({ id, name });
      this.recentProjects = items.slice(0, RECENT_PROJECTS_MAX);
      this._saveRecent(this.recentProjects);
    },

    enrichRecentFromProjects(projects) {
      const list = Array.isArray(projects) ? projects : [];
      if (!list.length || !this.recentProjects.length) return;
      const byId = new Map(list.map((p) => [String(p.id), String(p.name || p.id)]));
      let changed = false;
      this.recentProjects = this.recentProjects.map((item) => {
        const name = byId.get(item.id);
        if (name && name !== item.name) {
          changed = true;
          return { ...item, name };
        }
        return item;
      });
      if (changed) this._saveRecent(this.recentProjects);
    },

    projectTasksHref(id) {
      const slug = encodeURIComponent(String(id || "").trim());
      return `/#/projects/${slug}/tasks`;
    },

    projectSlugFromPath(path) {
      return window.parseProjectSlugFromPath(path || this.path);
    },

    isActive(href) {
      const p = String(this.path || "/").split("?")[0];
      const target = String(href || "/");
      if (target === "/dashboards/personal") return p === "/dashboards/personal";
      if (target === "/dashboards/tasks") {
        if (p !== "/dashboards/tasks" && p !== "/dashboards") return false;
        try {
          const full = String(this.path || "");
          const qIdx = full.indexOf("?");
          if (qIdx >= 0) {
            const qs = new URLSearchParams(full.slice(qIdx + 1));
            if (qs.get("project")) return false;
          }
        } catch {
          console.debug("nav.isActive: failed to parse dashboards/tasks query");
        }
        return true;
      }
      if (target === "/projects") return p === "/projects";
      if (target === "/admin") return p === "/admin" || p.startsWith("/admin/");
      return p === target;
    },

    isRecentProjectActive(id) {
      const slug = String(id || "").trim();
      if (!slug) return false;
      const current = this.projectSlugFromPath(this.path);
      if (current === slug) return true;
      try {
        const full = String(this.path || "");
        const base = full.split("?")[0];
        if (base.startsWith("/tasks/new/")) {
          const part = decodeURIComponent(base.slice("/tasks/new/".length) || "").trim();
          return part === slug;
        }
      } catch {
        console.debug("nav.isRecentProjectActive: failed to parse path");
      }
      return false;
    },

    trackPath(path) {
      const full = String(path || this.path || "");
      const slug = window.parseProjectSlugFromPath(full);
      if (!slug) return;
      const recent = (this.recentProjects || []).find((p) => p && p.id === slug);
      this.touchProject({ id: slug, name: (recent && recent.name) || slug });
    },

    _loadRecent() {
      try {
        const raw = localStorage.getItem(RECENT_PROJECTS_KEY);
        if (!raw) return [];
        const parsed = JSON.parse(raw);
        if (!Array.isArray(parsed)) return [];
        return parsed
          .map((p) => ({
            id: String((p && p.id) || "").trim(),
            name: String((p && p.name) || "").trim(),
          }))
          .filter((p) => p.id)
          .slice(0, RECENT_PROJECTS_MAX);
      } catch {
        return [];
      }
    },

    _saveRecent(items) {
      try {
        localStorage.setItem(RECENT_PROJECTS_KEY, JSON.stringify(items || []));
      } catch {
        console.debug("nav: localStorage.setItem(recent projects) failed");
      }
    },
  });

  Alpine.store("nav").init();
});

window.recordRecentProject = function recordRecentProject(id, name) {
  try {
    if (window.Alpine && Alpine.store("nav")) {
      Alpine.store("nav").touchProject({ id, name: name || id });
    }
  } catch {
    console.debug("recordRecentProject failed");
  }
};

window.parseProjectSlugFromPath = function parseProjectSlugFromPath(path) {
  try {
    let full = String(path || "");
    if (full.startsWith("#")) full = full.slice(1);
    if (full && !full.startsWith("/")) full = `/${full}`;
    const qIdx = full.indexOf("?");
    const qs = new URLSearchParams(qIdx >= 0 ? full.slice(qIdx + 1) : "");
    const fromQuery = qs.get("project");
    if (fromQuery) return String(fromQuery).trim();

    const base = (qIdx >= 0 ? full.slice(0, qIdx) : full).split("?")[0];
    const match = base.match(/^\/projects\/([^/]+)\/tasks$/);
    if (match) return decodeURIComponent(match[1] || "").trim();

    if (base.startsWith("/tasks/new/")) {
      return decodeURIComponent(base.slice("/tasks/new/".length) || "").trim();
    }
  } catch {
    console.debug("parseProjectSlugFromPath failed");
  }
  return "";
};

window.projectTasksLocation = function projectTasksLocation(slug) {
  const id = String(slug || "").trim();
  if (!id) return "/#/projects";
  return `/#/projects/${encodeURIComponent(id)}/tasks`;
};

window.touchProjectFromSlug = function touchProjectFromSlug(slug, name) {
  const id = String(slug || "").trim();
  if (!id) return;
  try {
    if (window.Alpine && Alpine.store("nav")) {
      Alpine.store("nav").touchProject({ id, name: String(name || id).trim() || id });
    }
  } catch {
    console.debug("touchProjectFromSlug failed");
  }
};

window.loginPage = function loginPage() {
  return {
    form: { email: "", password: "" },
    loading: false,
    error: "",
    notice: "",

    init() {
      // If already logged in, go to dashboard
      if (Alpine.store("auth").token) {
        window.location.assign("/#/dashboards/personal");
      }

      try {
        const url = new URL(window.location.href);
        const registeredFlag = url.searchParams.get("registered");
        if (registeredFlag === "1") {
          this.notice = "Аккаунт создан. Он ожидает активации администратором.";
        }
      } catch {
        console.debug("loginPage: failed to parse URL params");
      }
    },

    async submit() {
      this.error = "";
      this.notice = "";
      this.loading = true;
      try {
        await Alpine.store("auth").login(this.form.email, this.form.password);
        window.location.assign("/#/dashboards/personal");
      } catch (err) {
        if (err && err.status === 403) {
          window.location.assign("/#/pending");
          return;
        }
        toastApiError(err, "Не удалось войти");
      } finally {
        this.loading = false;
      }
    },
  };
};

window.registerPage = function registerPage() {
  return {
    form: { email: "", password: "", password2: "" },
    loading: false,
    error: "",

    init() {
      if (Alpine.store("auth").token) {
        window.location.assign("/#/dashboards/personal");
      }
    },

    async submit() {
      this.error = "";

      if (this.form.password.length < 8) {
        this.error = "Пароль должен быть не короче 8 символов";
        return;
      }
      if (this.form.password.length > 72) {
        this.error = "Пароль не длиннее 72 символов";
        return;
      }
      if (this.form.password !== this.form.password2) {
        this.error = "Пароли не совпадают";
        return;
      }

      this.loading = true;
      try {
        await Alpine.store("auth").register(this.form.email, this.form.password);
        window.location.assign("/#/pending");
      } catch (err) {
        toastApiError(err, "Не удалось зарегистрироваться");
      } finally {
        this.loading = false;
      }
    },
  };
};

window.dashboardPage = function dashboardPage() {
  return {
    projects: [],
    loading: false,
    error: "",

    init() {
      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }
      this.load();
    },

    async load() {
      this.error = "";
      this.loading = true;
      try {
        const { data } = await window.apiFetch("/projects");
        this.projects = (data && data.items) || [];
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить проекты");
      } finally {
        this.loading = false;
      }
    },
  };
};

let __projectsTaskCreatedHandler = null;

window.projectsPage = function projectsPage() {
  return {
    projects: [],
    total: 0,
    loading: false,
    error: "",
    notice: "",
    filter: {
      limit: 20,
      offset: 0,
    },

    safeRepoUrl(raw) {
      const s = String(raw == null ? "" : raw).trim();
      if (!s) return "";
      try {
        const u = new URL(s);
        const proto = (u.protocol || "").toLowerCase();
        if (proto !== "http:" && proto !== "https:") return "";
        if (u.username || u.password) return "";
        return u.toString();
      } catch {
        return "";
      }
    },

    init() {
      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }
      if (__projectsTaskCreatedHandler) {
        window.removeEventListener("task-created", __projectsTaskCreatedHandler);
      }
      __projectsTaskCreatedHandler = (e) => {
        try {
          const slug = e && e.detail && e.detail.projectSlug ? String(e.detail.projectSlug) : "";
          this.notice = slug ? `Задача создана (${slug})` : "Задача создана";
          window.setTimeout(() => {
            if (this.notice) this.notice = "";
          }, 3000);
        } catch {
          this.notice = "Задача создана";
        }
      };
      window.addEventListener("task-created", __projectsTaskCreatedHandler);
      this.load();
    },

    hasPrev() {
      const offset = Number(this.filter && this.filter.offset != null ? this.filter.offset : 0);
      return Number.isFinite(offset) && offset > 0;
    },

    hasNext() {
      const total = Number(this.total != null ? this.total : 0);
      const limit = Number(this.filter && this.filter.limit != null ? this.filter.limit : 20);
      const offset = Number(this.filter && this.filter.offset != null ? this.filter.offset : 0);
      if (!Number.isFinite(total) || !Number.isFinite(limit) || !Number.isFinite(offset)) return false;
      if (total <= 0) return false;
      return offset + limit < total;
    },

    _queryString() {
      const params = new URLSearchParams();
      params.set("limit", String(this.filter.limit || 20));
      params.set("offset", String(this.filter.offset || 0));
      const queryString = params.toString();
      return queryString ? `?${queryString}` : "";
    },

    async load() {
      this.error = "";
      this.loading = true;
      try {
        const { data } = await window.apiFetch(`/projects${this._queryString()}`);
        this.projects = (data && data.items) || [];
        this.total = (data && data.total) || 0;
        try {
          Alpine.store("nav").enrichRecentFromProjects(this.projects);
        } catch {
          console.debug("projectsPage.load: enrichRecentFromProjects failed");
        }
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить проекты");
      } finally {
        this.loading = false;
      }
    },

    nextPage() {
      const next = (this.filter.offset || 0) + (this.filter.limit || 20);
      if (next >= this.total) return;
      this.filter.offset = next;
      this.load();
    },

    prevPage() {
      const prev = (this.filter.offset || 0) - (this.filter.limit || 20);
      this.filter.offset = Math.max(0, prev);
      this.load();
    },

    projectTasksHref(project) {
      const id = project && project.id != null ? project.id : project;
      try {
        if (window.Alpine && Alpine.store("nav")) {
          return Alpine.store("nav").projectTasksHref(id);
        }
      } catch {
        console.debug("projectsPage.projectTasksHref: nav store unavailable");
      }
      return window.projectTasksLocation(id);
    },

    openProject(project) {
      if (!project || !project.id) return;
      try {
        Alpine.store("nav").touchProject({ id: project.id, name: project.name || project.id });
      } catch {
        console.debug("projectsPage.openProject: touchProject failed");
      }
      const href = this.projectTasksHref(project);
      if (typeof window.__spaNavigate === "function") {
        window.__spaNavigate(href);
        return;
      }
      const hashIdx = href.indexOf("#");
      window.location.hash = hashIdx >= 0 ? href.slice(hashIdx + 1) : href;
    },
  };
};

const TASK_STATUSES = ["New", "Open", "In Progress", "Resolved", "Closed", "Reopened"];
const DEFAULT_STATUS_FILTERS = ["New", "Open", "In Progress", "Reopened"];

window.TASK_STATUSES = TASK_STATUSES;

window.projectTasksPage = function projectTasksPage() {
  return {
    projectSlug: "",
    projectName: "",
    project: null,
    items: [],
    total: 0,
    limit: 20,
    offset: 0,
    loading: false,
    statusFilters: [...DEFAULT_STATUS_FILTERS],
    sort: "-created_at",
    error: "",

    safeRepoUrl(raw) {
      const s = String(raw == null ? "" : raw).trim();
      if (!s) return "";
      try {
        const u = new URL(s);
        const proto = (u.protocol || "").toLowerCase();
        if (proto !== "http:" && proto !== "https:") return "";
        if (u.username || u.password) return "";
        return u.toString();
      } catch {
        return "";
      }
    },

    toggleStatus(status) {
      const idx = this.statusFilters.indexOf(status);
      if (idx >= 0) this.statusFilters.splice(idx, 1);
      else this.statusFilters.push(status);
      this.offset = 0;
      this.loadTasks();
    },

    isStatusActive(status) {
      return this.statusFilters.includes(status);
    },

    hasActiveFilters() {
      if (this.statusFilters.length !== DEFAULT_STATUS_FILTERS.length) return true;
      return DEFAULT_STATUS_FILTERS.some(s => !this.statusFilters.includes(s));
    },

    clearFilters() {
      this.statusFilters = [...DEFAULT_STATUS_FILTERS];
      this.offset = 0;
      this.loadTasks();
    },

    setSort(value) {
      this.sort = value;
      this.offset = 0;
      this.loadTasks();
    },

    init() {
      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }
      this.projectSlug = window.parseProjectSlugFromPath(window.location.hash || "");
      if (!this.projectSlug) {
        window.location.assign("/#/projects");
        return;
      }
      this.loadProject();
      this.loadTasks();
    },

    async loadProject() {
      try {
        const { data } = await window.apiFetch(`/projects/${encodeURIComponent(this.projectSlug)}`);
        this.project = data || null;
        const name = data && (data.name || data.id);
        if (name) {
          this.projectName = String(name);
          window.recordRecentProject(this.projectSlug, this.projectName);
        }
      } catch (err) {
        if (err && err.status === 404) {
          window.location.assign("/#/projects");
          return;
        }
        if (err && err.status !== 401) {
          toastApiError(err, "Не удалось загрузить проект");
        }
      }
    },

    hasPrev() {
      return Number.isFinite(this.offset) && this.offset > 0;
    },

    hasNext() {
      if (!Number.isFinite(this.total) || this.total <= 0) return false;
      return this.offset + this.limit < this.total;
    },

    nextPage() {
      const next = (this.offset || 0) + (this.limit || 20);
      if (next >= this.total) return;
      this.offset = next;
      this.loadTasks();
    },

    prevPage() {
      this.offset = Math.max(0, (this.offset || 0) - (this.limit || 20));
      this.loadTasks();
    },

    async loadTasks() {
      this.error = "";
      this.loading = true;
      try {
        const params = new URLSearchParams();
        params.set("project", this.projectSlug);
        params.set("limit", String(this.limit || 20));
        params.set("offset", String(this.offset || 0));
        if (this.statusFilters.length > 0 && this.statusFilters.length < TASK_STATUSES.length) {
          params.set("statuses", this.statusFilters.join(","));
        }
        params.set("sort", this.sort);

        const { data } = await window.apiFetch(`/tasks?${params.toString()}`);
        this.items = (data && data.items) || [];
        this.total = (data && data.total) || 0;
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить задачи");
      } finally {
        this.loading = false;
      }
    },

    paginationText() {
      const total = Number(this.total != null ? this.total : 0);
      const limit = Number(this.limit != null ? this.limit : 20);
      const offset = Number(this.offset != null ? this.offset : 0);
      if (!Number.isFinite(total) || total <= 0) return "0 задач";
      const from = Math.min(total, offset + 1);
      const to = Math.min(total, offset + limit);
      return `${from}–${to} из ${total}`;
    },

    formatDate(iso) {
      if (!iso) return "";
      try {
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return String(iso);
        return d.toLocaleString("ru-RU", { year: "numeric", month: "2-digit", day: "2-digit" });
      } catch {
        return String(iso);
      }
    },
  };
};

window.adminHomePage = function adminHomePage() {
  return {
    init() {
      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }
      const currentUser = Alpine.store("auth").user;
      if (!currentUser || currentUser.role !== "admin") {
        window.location.assign("/#/dashboards/personal");
        return;
      }
    },
  };
};

window.adminUsersPage = function adminUsersPage() {
  return {
    users: [],
    total: 0,
    loading: false,
    error: "",
    filter: {
      status: "",
      role: "",
      limit: 20,
      offset: 0,
    },

    hasPrev() {
      const offset = Number(this.filter && this.filter.offset != null ? this.filter.offset : 0);
      return Number.isFinite(offset) && offset > 0;
    },

    hasNext() {
      const total = Number(this.total != null ? this.total : 0);
      const limit = Number(this.filter && this.filter.limit != null ? this.filter.limit : 20);
      const offset = Number(this.filter && this.filter.offset != null ? this.filter.offset : 0);
      if (!Number.isFinite(total) || !Number.isFinite(limit) || !Number.isFinite(offset)) return false;
      if (total <= 0) return false;
      return offset + limit < total;
    },

    init() {
      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }
      const currentUser = Alpine.store("auth").user;
      if (!currentUser || currentUser.role !== "admin") {
        window.location.assign("/#/dashboards/personal");
        return;
      }
      this.load();
    },

    _queryString() {
      const params = new URLSearchParams();
      if (this.filter.status) params.set("status", this.filter.status);
      if (this.filter.role) params.set("role", this.filter.role);
      params.set("limit", String(this.filter.limit || 20));
      params.set("offset", String(this.filter.offset || 0));
      const queryString = params.toString();
      return queryString ? `?${queryString}` : "";
    },

    async load() {
      this.error = "";
      this.loading = true;
      try {
        const { data } = await window.apiFetch(`/users${this._queryString()}`);
        this.users = (data && data.items) || [];
        this.total = (data && data.total) || 0;
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить пользователей");
      } finally {
        this.loading = false;
      }
    },

    applyFilters() {
      this.filter.offset = 0;
      this.load();
    },

    nextPage() {
      const next = (this.filter.offset || 0) + (this.filter.limit || 20);
      if (next >= this.total) return;
      this.filter.offset = next;
      this.load();
    },

    prevPage() {
      const prev = (this.filter.offset || 0) - (this.filter.limit || 20);
      this.filter.offset = Math.max(0, prev);
      this.load();
    },

    async updateUser(id, patch) {
      this.error = "";
      this.loading = true;
      try {
        await window.apiFetch(`/users/${id}`, { method: "PATCH", body: patch });
        toastSuccess("Пользователь успешно обновлён");
        await this.load();
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось обновить пользователя");
      } finally {
        this.loading = false;
      }
    },
  };
};

window.adminProjectsPage = function adminProjectsPage() {
  return {
    projects: [],
    total: 0,
    loading: false,
    error: "",
    filter: {
      limit: 20,
      offset: 0,
    },
    safeRepoUrl(raw) {
      const s = String(raw == null ? "" : raw).trim();
      if (!s) return "";
      try {
        const u = new URL(s);
        const proto = (u.protocol || "").toLowerCase();
        if (proto !== "http:" && proto !== "https:") return "";
        if (u.username || u.password) return "";
        return u.toString();
      } catch {
        return "";
      }
    },
    deleteModal: {
      open: false,
      id: "",
      name: "",
      saving: false,
      error: "",
    },
    projectModal: {
      open: false,
      mode: "create",
      id: "",
      saving: false,
      error: "",
      original: null,
      form: { name: "", repo_url: "" },
    },

    init() {
      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }
      const currentUser = Alpine.store("auth").user;
      if (!currentUser || currentUser.role !== "admin") {
        window.location.assign("/#/dashboards/personal");
        return;
      }
      this.load();
    },

    hasPrev() {
      const offset = Number(this.filter && this.filter.offset != null ? this.filter.offset : 0);
      return Number.isFinite(offset) && offset > 0;
    },

    hasNext() {
      const total = Number(this.total != null ? this.total : 0);
      const limit = Number(this.filter && this.filter.limit != null ? this.filter.limit : 20);
      const offset = Number(this.filter && this.filter.offset != null ? this.filter.offset : 0);
      if (!Number.isFinite(total) || !Number.isFinite(limit) || !Number.isFinite(offset)) return false;
      if (total <= 0) return false;
      return offset + limit < total;
    },

    _queryString() {
      const params = new URLSearchParams();
      params.set("limit", String(this.filter.limit || 20));
      params.set("offset", String(this.filter.offset || 0));
      const queryString = params.toString();
      return queryString ? `?${queryString}` : "";
    },

    async load() {
      this.error = "";
      this.loading = true;
      try {
        const { data } = await window.apiFetch(`/projects${this._queryString()}`);
        this.projects = (data && data.items) || [];
        this.total = (data && data.total) || 0;
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить проекты");
      } finally {
        this.loading = false;
      }
    },

    nextPage() {
      const next = (this.filter.offset || 0) + (this.filter.limit || 20);
      if (next >= this.total) return;
      this.filter.offset = next;
      this.load();
    },

    prevPage() {
      const prev = (this.filter.offset || 0) - (this.filter.limit || 20);
      this.filter.offset = Math.max(0, prev);
      this.load();
    },

    openProjectModal(project = null) {
      const isEdit = Boolean(project);
      const id = isEdit ? (project.id || "") : "";

      this.projectModal.open = true;
      this.projectModal.mode = isEdit ? "edit" : "create";
      this.projectModal.id = id;
      this.projectModal.error = "";
      this.projectModal.saving = false;
      this.projectModal.original = isEdit ? project : null;
      this.projectModal.form = {
        name: (project && project.name) || "",
        repo_url: (project && project.repo_url) || "",
      };
    },

    closeProjectModal(force = false) {
      if (!force && this.projectModal.saving) return;
      this.projectModal.open = false;
      this.projectModal.mode = "create";
      this.projectModal.id = "";
      this.projectModal.error = "";
      this.projectModal.saving = false;
      this.projectModal.original = null;
      this.projectModal.form = { name: "", repo_url: "" };
    },

    async save() {
      this.projectModal.error = "";
      const mode = this.projectModal.mode;
      const id = this.projectModal.id;
      const name = (this.projectModal.form.name || "").trim();
      const repoUrl = (this.projectModal.form.repo_url || "").trim();

      if (!name) {
        this.projectModal.error = "Название не должно быть пустым";
        return;
      }
      if (!repoUrl) {
        this.projectModal.error = "Repo URL не должен быть пустым";
        return;
      }
      let normalizedRepoUrl;
      try {
        const u = new URL(repoUrl);
        const proto = (u.protocol || "").toLowerCase();
        if (proto !== "https:") {
          this.projectModal.error = "Repo URL должен использовать https://";
          return;
        }
        if (u.username || u.password) {
          this.projectModal.error = "Repo URL не должен содержать учётные данные";
          return;
        }
        normalizedRepoUrl = u.toString();
      } catch {
        this.projectModal.error = "Некорректный Repo URL";
        return;
      }

      this.projectModal.saving = true;
      try {
        if (mode === "create") {
          await window.apiFetch("/projects", { method: "POST", body: { name, repo_url: normalizedRepoUrl } });
          toastSuccess("Проект успешно создан");
        } else {
          if (!id) {
            this.projectModal.error = "Не удалось определить ID проекта";
            return;
          }

          const patch = {};
          if (!this.projectModal.original || name !== (this.projectModal.original.name || "")) {
            patch.name = name;
          }
          if (this.projectModal.original) {
            const originalRaw = String(this.projectModal.original.repo_url || "").trim();
            let originalNormalized = originalRaw;
            try {
              if (originalRaw) {
                const ou = new URL(originalRaw);
                const oproto = (ou.protocol || "").toLowerCase();
                if (oproto === "https:" && !ou.username && !ou.password) {
                  originalNormalized = ou.toString();
                }
              }
            } catch {
              // keep originalRaw
            }
            if (normalizedRepoUrl !== originalNormalized) {
              patch.repo_url = normalizedRepoUrl;
            }
          } else {
            patch.repo_url = normalizedRepoUrl;
          }

          if (Object.keys(patch).length === 0) {
            this.closeProjectModal();
            return;
          }

          await window.apiFetch(`/projects/${encodeURIComponent(id)}`, { method: "PATCH", body: patch });
          toastSuccess("Проект успешно сохранён");
        }

        await this.load();
        this.closeProjectModal(true);
      } catch (err) {
        if (err && err.status === 401) return;
        const defaultMessage = mode === "create" ? "Не удалось создать проект" : "Не удалось сохранить изменения";
        toastApiError(err, defaultMessage);
      } finally {
        this.projectModal.saving = false;
      }
    },

    openDeleteModal(project) {
      const id = (project && project.id) || "";
      const name = (project && project.name) || id;
      if (!id) {
        this.error = "Не удалось определить ID проекта";
        return;
      }

      this.deleteModal.open = true;
      this.deleteModal.id = id;
      this.deleteModal.name = name;
      this.deleteModal.error = "";
      this.deleteModal.saving = false;
    },

    closeDeleteModal(force = false) {
      if (!force && this.deleteModal.saving) return;
      this.deleteModal.open = false;
      this.deleteModal.id = "";
      this.deleteModal.name = "";
      this.deleteModal.error = "";
      this.deleteModal.saving = false;
    },

    async confirmDeleteModal() {
      this.deleteModal.error = "";
      const id = this.deleteModal.id;
      if (!id) {
        this.deleteModal.error = "Не удалось определить ID проекта";
        return;
      }

      this.deleteModal.saving = true;
      try {
        await window.apiFetch(`/projects/${encodeURIComponent(id)}`, { method: "DELETE" });
        toastSuccess("Проект успешно удалён");
        await this.load();
        this.closeDeleteModal(true);
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось удалить проект");
      } finally {
        this.deleteModal.saving = false;
      }
    },

  };
};

window.profilePage = function profilePage() {
  return {
    currentUser: { name: "", email: "", role: "", status: "" },
    form: { old_password: "", new_password: "", new_password2: "" },
    loading: false,
    error: "",
    success: "",

    init() {
      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }
      const currentUser = Alpine.store("auth").user || null;
      this.currentUser = {
        name: (currentUser && currentUser.name) || "",
        email: (currentUser && currentUser.email) || "",
        role: (currentUser && currentUser.role) || "",
        status: (currentUser && currentUser.status) || "",
      };
    },

    resetForm() {
      if (this.loading) return;
      this.error = "";
      this.success = "";
      this.form = { old_password: "", new_password: "", new_password2: "" };
    },

    async submit() {
      this.error = "";
      this.success = "";

      const oldPassword = this.form.old_password || "";
      const newPassword = this.form.new_password || "";
      const newPassword2 = this.form.new_password2 || "";

      if (!oldPassword) {
        this.error = "Введите текущий пароль";
        return;
      }
      if (newPassword.length < 8) {
        this.error = "Новый пароль должен быть не короче 8 символов";
        return;
      }
      if (newPassword.length > 72) {
        this.error = "Новый пароль не длиннее 72 символов";
        return;
      }
      if (newPassword !== newPassword2) {
        this.error = "Новые пароли не совпадают";
        return;
      }

      this.loading = true;
      try {
        await window.apiFetch("/auth/change-password", {
          method: "POST",
          body: { old_password: oldPassword, new_password: newPassword },
        });
        this.success = "Пароль изменён";
        toastSuccess("Пароль успешно изменён");
        this.form = { old_password: "", new_password: "", new_password2: "" };
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось сменить пароль");
      } finally {
        this.loading = false;
      }
    },
  };
};

window.dashboardsPage = function dashboardsPage(initialSection = "personal") {
  return {
    section: initialSection === "all" ? "all" : "personal",
    error: "",

    loading: {
      created: false,
      assigned: false,
      all: false,
    },

    statusFilters: [...DEFAULT_STATUS_FILTERS],
    sort: "-created_at",

    created: { items: [], total: 0, limit: 20, offset: 0 },
    assigned: { items: [], total: 0, limit: 20, offset: 0 },
    all: { items: [], total: 0, limit: 20, offset: 0 },

    toggleStatus(status) {
      const idx = this.statusFilters.indexOf(status);
      if (idx >= 0) this.statusFilters.splice(idx, 1);
      else this.statusFilters.push(status);
      this._resetOffsets();
      if (this.section === "personal") this.loadPersonal();
      else this.loadAll();
    },

    isStatusActive(status) {
      return this.statusFilters.includes(status);
    },

    hasActiveFilters() {
      if (this.statusFilters.length !== DEFAULT_STATUS_FILTERS.length) return true;
      return DEFAULT_STATUS_FILTERS.some(s => !this.statusFilters.includes(s));
    },

    clearFilters() {
      this.statusFilters = [...DEFAULT_STATUS_FILTERS];
      this._resetOffsets();
      if (this.section === "personal") this.loadPersonal();
      else this.loadAll();
    },

    setSort(value) {
      this.sort = value;
      this._resetOffsets();
      if (this.section === "personal") this.loadPersonal();
      else this.loadAll();
    },

    _resetOffsets() {
      this.created.offset = 0;
      this.assigned.offset = 0;
      this.all.offset = 0;
    },

    init() {
      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }
      this._initFromLocation();
      if (this.section === "personal") this.loadPersonal();
      else this.loadAll();
    },

    setSection(section) {
      if (section !== "personal" && section !== "all") return;
      if (this.section === section) return;
      this.section = section;
      if (section === "personal") this.loadPersonal();
      else this.loadAll();
    },

    _initFromLocation() {
      try {
        const url = new URL(window.location.href);
        const limit = Number(url.searchParams.get("limit"));
        if (Number.isFinite(limit) && limit > 0) {
          const v = Math.min(100, limit);
          this.created.limit = v;
          this.assigned.limit = v;
          this.all.limit = v;
        }
        const statuses = url.searchParams.get("statuses");
        if (statuses && statuses.trim()) {
          this.statusFilters = statuses.split(",").filter(s => TASK_STATUSES.includes(s));
        }
        const sort = url.searchParams.get("sort");
        if (sort && sort.trim()) {
          this.sort = sort;
        }
      } catch {
        console.debug("dashboardsPage: failed to parse URL params (limit/statuses)");
      }
    },

    _commonQueryParams() {
      const params = new URLSearchParams();
      try {
        const current = new URL(window.location.href);
        const allow = ["project", "statuses", "priorities"];
        for (const key of allow) {
          const v = current.searchParams.get(key);
          if (v != null && v !== "") params.set(key, v);
        }
      } catch {
        console.debug("dashboardsPage: failed to parse URL search params");
      }
      params.set("sort", this.sort);
      return params;
    },

    hasPrev(state) {
      const offset = Number(state && state.offset != null ? state.offset : 0);
      return Number.isFinite(offset) && offset > 0;
    },

    hasNext(state) {
      const total = Number(state && state.total != null ? state.total : 0);
      const limit = Number(state && state.limit != null ? state.limit : 20);
      const offset = Number(state && state.offset != null ? state.offset : 0);
      if (!Number.isFinite(total) || !Number.isFinite(limit) || !Number.isFinite(offset)) return false;
      if (total <= 0) return false;
      return offset + limit < total;
    },

    nextPage(which) {
      const state = which === "created" ? this.created : which === "assigned" ? this.assigned : this.all;
      const next = (state.offset || 0) + (state.limit || 20);
      if (next >= state.total) return;
      state.offset = next;
      if (which === "created") this.loadCreated();
      else if (which === "assigned") this.loadAssigned();
      else this.loadAll();
    },

    prevPage(which) {
      const state = which === "created" ? this.created : which === "assigned" ? this.assigned : this.all;
      const prev = (state.offset || 0) - (state.limit || 20);
      state.offset = Math.max(0, prev);
      if (which === "created") this.loadCreated();
      else if (which === "assigned") this.loadAssigned();
      else this.loadAll();
    },

    async loadPersonal() {
      await Promise.all([this.loadCreated(), this.loadAssigned()]);
    },

    async loadCreated() {
      this.error = "";
      this.loading.created = true;
      try {
        const user = Alpine.store("auth").user;
        const userId = user && user.id;
        if (!userId) throw new Error("Не удалось определить пользователя");

        const params = this._commonQueryParams();
        params.set("author", String(userId));
        params.set("limit", String(this.created.limit || 20));
        params.set("offset", String(this.created.offset || 0));
        if (this.statusFilters.length > 0 && this.statusFilters.length < TASK_STATUSES.length) {
          params.set("statuses", this.statusFilters.join(","));
        }

        const { data } = await window.apiFetch(`/tasks?${params.toString()}`);
        this.created.items = (data && data.items) || [];
        this.created.total = (data && data.total) || 0;
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить созданные задачи");
      } finally {
        this.loading.created = false;
      }
    },

    async loadAssigned() {
      this.error = "";
      this.loading.assigned = true;
      try {
        const user = Alpine.store("auth").user;
        const userId = user && user.id;
        if (!userId) throw new Error("Не удалось определить пользователя");

        const params = this._commonQueryParams();
        params.set("assignee", String(userId));
        params.set("limit", String(this.assigned.limit || 20));
        params.set("offset", String(this.assigned.offset || 0));
        if (this.statusFilters.length > 0 && this.statusFilters.length < TASK_STATUSES.length) {
          params.set("statuses", this.statusFilters.join(","));
        }

        const { data } = await window.apiFetch(`/tasks?${params.toString()}`);
        this.assigned.items = (data && data.items) || [];
        this.assigned.total = (data && data.total) || 0;
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить назначенные задачи");
      } finally {
        this.loading.assigned = false;
      }
    },

    async loadAll() {
      this.error = "";
      this.loading.all = true;
      try {
        const params = this._commonQueryParams();
        params.set("limit", String(this.all.limit || 20));
        params.set("offset", String(this.all.offset || 0));
        if (this.statusFilters.length > 0 && this.statusFilters.length < TASK_STATUSES.length) {
          params.set("statuses", this.statusFilters.join(","));
        }

        const { data } = await window.apiFetch(`/tasks?${params.toString()}`);
        this.all.items = (data && data.items) || [];
        this.all.total = (data && data.total) || 0;
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить задачи");
      } finally {
        this.loading.all = false;
      }
    },

    paginationText(state) {
      const total = Number(state && state.total != null ? state.total : 0);
      const limit = Number(state && state.limit != null ? state.limit : 20);
      const offset = Number(state && state.offset != null ? state.offset : 0);
      if (!Number.isFinite(total) || total <= 0) return "0 задач";
      const from = Math.min(total, offset + 1);
      const to = Math.min(total, offset + limit);
      return `${from}–${to} из ${total}`;
    },

    formatDate(iso) {
      if (!iso) return "";
      try {
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return String(iso);
        return d.toLocaleString("ru-RU", { year: "numeric", month: "2-digit", day: "2-digit" });
      } catch {
        return String(iso);
      }
    },

    projectTasksHref(slug) {
      return window.projectTasksLocation(slug);
    },

    onProjectLinkClick(slug) {
      window.touchProjectFromSlug(slug);
    },
  };
};

window.taskPage = function taskPage(taskIdRaw, modeRaw = "view", projectSlugRaw = "") {
  return {
    taskId: String(taskIdRaw || "").trim(),
    mode: String(modeRaw || "view").trim(),
    projectSlug: String(projectSlugRaw || "").trim(),
    task: null,
    selectedAssignee: null,
    comments: [],
    attachments: [],
    loading: false,
    error: "",
    notice: "",
    editor: {
      loading: false,
      saving: false,
      error: "",
      original: null,
      form: {
        project_slug: "",
        title: "",
        description: "",
        priority: "Minor",
        status: "New",
        kind: "Task",
        assignee_id: "",
        due_date: "",
      },
    },
    assigneeSearch: {
      query: "",
      items: [],
      total: 0,
      loading: false,
      error: "",
      _debounceTimer: null,
      _requestSeq: 0,
    },
    assigneeSave: {
      saving: false,
      error: "",
    },
    attachmentUpload: {
      files: [],
      fileLabel: "Файл не выбран",
      uploading: false,
      error: "",
    },
    deletingAttachmentId: null,
    attachmentDelete: {
      error: "",
    },
    attachmentDeleteModal: {
      open: false,
      attachmentId: null,
      fileName: "",
      saving: false,
      error: "",
    },
    commentForm: {
      content: "",
      saving: false,
      error: "",
    },
    editingCommentId: null,
    editingContent: "",
    commentEdit: {
      saving: false,
      error: "",
    },
    deletingCommentId: null,
    commentDelete: {
      saving: false,
      error: "",
    },
    deleteModal: {
      open: false,
      commentId: null,
      preview: "",
      saving: false,
      error: "",
    },
    statusEdit: {
      pending: "",
      comment: "",
      saving: false,
      error: "",
    },

    safeRepoUrl(raw) {
      const s = String(raw == null ? "" : raw).trim();
      if (!s) return "";
      try {
        const u = new URL(s);
        const proto = (u.protocol || "").toLowerCase();
        if (proto !== "http:" && proto !== "https:") return "";
        if (u.username || u.password) return "";
        return u.toString();
      } catch {
        return "";
      }
    },

    renderMarkdown(text) {
      if (typeof window.renderMarkdown === "function") {
        return window.renderMarkdown(text);
      }
      const s = String(text ?? "").trim();
      return s ? `<p class="whitespace-pre-wrap">${s}</p>` : '<p class="text-slate-500">—</p>';
    },

    _markdownFieldValue(selector, fallback = "") {
      if (typeof window.getMarkdownTextareaValue === "function") {
        const value = window.getMarkdownTextareaValue(selector);
        if (value != null) return String(value).trim();
      }
      return String(fallback ?? "").trim();
    },

    destroyMarkdownEditors() {
      if (typeof window.destroyAllMarkdownEditors === "function") {
        window.destroyAllMarkdownEditors();
      }
    },

    refreshMarkdownEditors() {
      this.destroyMarkdownEditors();
      this.$nextTick(() => {
        if (typeof window.isMarkdownEditorReady !== "function" || !window.isMarkdownEditorReady()) {
          return;
        }
        if (typeof window.initMarkdownEditor !== "function") return;

        const descriptionOpts = {
          mode: "plain",
          placeholder: "Можно Markdown",
          label: "Описание задачи",
          maxHeight: 480,
          toolbar: [
            "undo",
            "redo",
            "heading",
            "blockquote",
            "ul",
            "ol",
            "checklist",
            "bold",
            "italic",
            "strikethrough",
            "code",
            "codeblock",
            "link",
            "image",
            "hr",
            "table",
            "preview",
          ],
        };

        const commentOpts = {
          mode: "plain",
          placeholder: "Напишите комментарий…",
          label: "Новый комментарий",
          maxHeight: 240,
          footer: false,
          toolbar: [
            "undo",
            "redo",
            "bold",
            "italic",
            "code",
            "link",
            "ul",
            "ol",
            "checklist",
            "preview",
          ],
        };

        const commentEditOpts = {
          mode: "plain",
          label: "Редактировать комментарий",
          maxHeight: 240,
          footer: false,
          toolbar: commentOpts.toolbar,
        };

        if (!this.loading && (this.isCreate() || this.isEdit())) {
          window.initMarkdownEditor("#task-description", descriptionOpts);
        }

        if (!this.isCreate() && !this.loading) {
          window.initMarkdownEditor("#task-new-comment", commentOpts);
        }

        if (this.editingCommentId) {
          window.initMarkdownEditor(`#task-comment-edit-${this.editingCommentId}`, commentEditOpts);
        }
      });
    },

    init() {
      this.destroyMarkdownEditors();
      try {
        if (this.assigneeSave) this.assigneeSave.saving = false;
        if (this.assigneeSave) this.assigneeSave.error = "";
        if (this.assigneeSearch) this.assigneeSearch.loading = false;
        if (this.assigneeSearch) this.assigneeSearch.error = "";
        if (this.assigneeSearch && this.assigneeSearch._debounceTimer) {
          window.clearTimeout(this.assigneeSearch._debounceTimer);
          this.assigneeSearch._debounceTimer = null;
        }
      } catch {
        console.debug("taskPage.init: failed to reset local edit state");
      }

      if (!Alpine.store("auth").token) {
        window.location.assign("/#/login");
        return;
      }

      this._normalizeModeFromRoute();

      if (this.mode === "create") {
        this._initCreateForm();
        try {
          document.title = "Новая задача — Bit Issues";
        } catch {
          console.debug("taskPage.init: failed to set document.title (create mode)");
        }
        this.$nextTick(() => this.refreshMarkdownEditors());
        return;
      }

      if (!this.taskId) {
        this.error = "Не удалось определить задачу";
        return;
      }
      this.load();
    },

    isView() {
      return this.mode === "view";
    },
    isEdit() {
      return this.mode === "edit";
    },
    isCreate() {
      return this.mode === "create";
    },

    _normalizeModeFromRoute() {
      const m = (this.mode || "").toLowerCase();
      if (m === "edit" || m === "create" || m === "view") {
        this.mode = m;
        return;
      }
      this.mode = "view";
    },

    _initCreateForm() {
      this.notice = "";
      this.editor.error = "";
      this.editor.original = null;
      this.editor.form = {
        project_slug: (this.projectSlug || "").trim(),
        title: "",
        description: "",
        priority: "Minor",
        status: "New",
        kind: "Task",
        assignee_id: "",
        due_date: "",
      };
      this.selectedAssignee = null;
      this.assigneeSearch.query = "";
      this.assigneeSearch.items = [];
      this.assigneeSearch.total = 0;
      this.assigneeSearch.loading = false;
      this.assigneeSearch.error = "";
    },

    goBack() {
      try {
        if (window.history && window.history.length > 1) {
          window.history.back();
          return;
        }
      } catch {
        console.debug("taskPage.goBack: history.back() failed");
      }
      const slug =
        this.projectSlug ||
        (this.task && this.task.project_slug) ||
        window.parseProjectSlugFromPath(window.location.hash || "");
      if (slug) {
        window.location.assign(window.projectTasksLocation(slug));
        return;
      }
      window.location.assign("/#/dashboards/tasks");
    },

    displayKey(t) {
      if (!t) return "";
      const slug = t.project_slug || "";
      const num = t.number != null ? String(t.number) : "";
      if (slug && num) return `${slug} #${num}`;
      if (num) return `#${num}`;
      return `task #${t.id || ""}`.trim();
    },

    projectTasksHref(slug) {
      return window.projectTasksLocation(slug);
    },

    onProjectLinkClick(slug) {
      window.touchProjectFromSlug(slug);
    },

    formatDateTime(iso) {
      if (!iso) return "";
      try {
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return String(iso);
        return d.toLocaleString("ru-RU", {
          year: "numeric",
          month: "2-digit",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
        });
      } catch {
        return String(iso);
      }
    },

    async load() {
      this.error = "";
      this.loading = true;
      try {
        const { data } = await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}`);
        this.task = data || null;
        this.comments = (data && data.comments) || [];
        this.attachments = (data && data.attachments) || [];
        this.assigneeSave.error = "";
        this.attachmentUpload.error = "";
        this.deletingAttachmentId = null;
        this.attachmentDelete.error = "";
        this.cancelEditComment(true);
        this.closeDeleteModal(true);

        this.statusEdit.pending = data ? (data.status || "") : "";
        this.statusEdit.comment = "";
        this.statusEdit.error = "";

        if (this.mode === "edit") {
          this._initEditFormFromTask(this.task);
        }

        if (data && data.title) {
          try {
            document.title = `${data.title} — Bit Issues`;
          } catch {
            console.debug("taskPage.load: failed to set document.title (task title)");
          }
        }

        if (data && data.project_slug) {
          window.recordRecentProject(data.project_slug, data.project_slug);
        }
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось загрузить задачу");
      } finally {
        this.loading = false;
        this.$nextTick(() => this.refreshMarkdownEditors());
      }
    },

    _initEditFormFromTask(t) {
      if (!t) return;
      this.editor.error = "";
      this.editor.original = {
        title: t.title || "",
        description: t.description || "",
        priority: t.priority || "Minor",
        status: t.status || "New",
        kind: t.kind || "Task",
        due_date: t.due_date || "",
        assignee_id: (t.assignee && t.assignee.id != null) ? String(t.assignee.id) : "",
        project_slug: t.project_slug || "",
      };
      this.editor.form = {
        project_slug: t.project_slug || "",
        title: t.title || "",
        description: t.description || "",
        priority: t.priority || "Minor",
        status: t.status || "New",
        kind: t.kind || "Task",
        assignee_id: (t.assignee && t.assignee.id != null) ? String(t.assignee.id) : "",
        due_date: t.due_date || "",
      };
      this.selectedAssignee = t.assignee || null;
      this.$nextTick(() => {
        try {
          const el = this.$refs && this.$refs.editorTitle;
          if (el && typeof el.focus === "function") el.focus();
        } catch {
          console.debug("taskPage._initEditFormFromTask: failed to focus editor title input");
        }
      });
    },

    startEdit() {
      if (!this.taskId) return;
      window.location.assign(`/#/tasks/${encodeURIComponent(this.taskId)}/edit`);
    },

    cancelEdit() {
      if (!this.taskId) return;
      window.location.assign(`/#/tasks/${encodeURIComponent(this.taskId)}`);
    },

    _assigneeIdPayload() {
      const raw = (this.editor.form.assignee_id || "").trim();
      if (!raw) return null;
      const v = Number(raw);
      if (!Number.isFinite(v) || v <= 0) throw new Error("assignee_id должен быть числом больше 0");
      return v;
    },

    _createPayload() {
      const projectSlug = (this.editor.form.project_slug || "").trim();
      const title = (this.editor.form.title || "").trim();
      const description = this._markdownFieldValue("#task-description", this.editor.form.description);
      const priority = (this.editor.form.priority || "").trim();
      const kind = (this.editor.form.kind || "").trim();
      const dueDate = (this.editor.form.due_date || "").trim();

      if (!projectSlug) throw new Error("Введите проект");
      if (!title) throw new Error("Введите заголовок");

      const body = {
        project_slug: projectSlug,
        title,
        description: description || "",
        priority: priority || "Minor",
        kind: kind || "Task",
      };

      const assigneeId = this._assigneeIdPayload();
      if (assigneeId != null) body.assignee_id = assigneeId;
      if (dueDate) body.due_date = dueDate;
      return body;
    },

    _patchPayload() {
      const title = (this.editor.form.title || "").trim();
      const description = this._markdownFieldValue("#task-description", this.editor.form.description);
      const priority = (this.editor.form.priority || "").trim();
      const status = (this.editor.form.status || "").trim();
      const kind = (this.editor.form.kind || "").trim();
      const dueDate = (this.editor.form.due_date || "").trim();

      if (!title) throw new Error("Заголовок не должен быть пустым");

      const patch = {};
      const orig = this.editor.original || {};

      if (title !== String(orig.title || "")) patch.title = title;
      if (description !== String(orig.description || "")) patch.description = description;
      if ((priority || "Minor") !== String(orig.priority || "Minor")) patch.priority = priority || "Minor";
      if ((status || "New") !== String(orig.status || "New")) patch.status = status || "New";
      if ((kind || "Task") !== String(orig.kind || "Task")) patch.kind = kind || "Task";

      const origDue = String(orig.due_date || "");
      if ((dueDate || "") !== origDue) patch.due_date = dueDate || "";

      const assigneeId = this._assigneeIdPayload();
      const origAssignee = (orig.assignee_id || "").trim();
      if (String(assigneeId || "") !== String(origAssignee || "")) {
        patch.assignee_id = assigneeId != null ? assigneeId : 0;
      }

      return patch;
    },

    async submitEditor() {
      this.editor.error = "";
      this.notice = "";
      if (this.editor.saving) return;

      this.editor.saving = true;
      try {
        if (this.mode === "create") {
          const body = this._createPayload();
          const { data } = await window.apiFetch("/tasks", { method: "POST", body });

          window.dispatchEvent(
            new CustomEvent("task-created", {
              detail: { projectSlug: body.project_slug },
            }),
          );
          window.recordRecentProject(body.project_slug, body.project_slug);
          toastSuccess("Задача создана");

          const id = data && data.id != null ? data.id : null;
          if (id) {
            window.location.assign(`/#/tasks/${encodeURIComponent(id)}`);
            return;
          }
          const slug = body.project_slug || this.projectSlug;
          window.location.assign(slug ? window.projectTasksLocation(slug) : "/#/dashboards/tasks");
          return;
        }

        const patch = this._patchPayload();
        if (Object.keys(patch).length === 0) {
          this.notice = "Изменений нет";
          return;
        }

        await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}`, { method: "PATCH", body: patch });
        toastSuccess("Задача успешно сохранена");
        window.location.assign(`/#/tasks/${encodeURIComponent(this.taskId)}`);
      } catch (err) {
        if (err && err.status === 401) return;
        if (err && err.status) {
          toastApiError(err, "Не удалось сохранить");
        } else {
          this.editor.error = err && err.message ? err.message : "Не удалось сохранить";
        }
      } finally {
        this.editor.saving = false;
      }
    },

    onAssigneeSearchInput() {
      const q = (this.assigneeSearch.query || "").trim();
      this.assigneeSearch.error = "";
      this.assigneeSearch.total = 0;

      if (this.assigneeSearch._debounceTimer) {
        window.clearTimeout(this.assigneeSearch._debounceTimer);
      }

      if (!q) {
        this.assigneeSearch.items = [];
        this.assigneeSearch.loading = false;
        return;
      }

      this.assigneeSearch._debounceTimer = window.setTimeout(() => {
        this.searchAssignees();
      }, 250);
    },

    async searchAssignees() {
      const q = (this.assigneeSearch.query || "").trim();
      if (!q) return;

      const seq = ++this.assigneeSearch._requestSeq;
      this.assigneeSearch.loading = true;
      this.assigneeSearch.error = "";
      try {
        const params = new URLSearchParams();
        params.set("query", q);
        params.set("limit", "20");
        params.set("offset", "0");

        const { data } = await window.apiFetch(`/users/search?${params.toString()}`);
        if (seq !== this.assigneeSearch._requestSeq) return;
        if (q !== (this.assigneeSearch.query || "").trim()) return;
        this.assigneeSearch.items = (data && data.items) || [];
        this.assigneeSearch.total = (data && data.total) || 0;
      } catch (err) {
        if (seq !== this.assigneeSearch._requestSeq) return;
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось выполнить поиск");
        this.assigneeSearch.items = [];
        this.assigneeSearch.total = 0;
      } finally {
        if (seq === this.assigneeSearch._requestSeq) {
          this.assigneeSearch.loading = false;
        }
      }
    },

    async setAssignee(userId) {
      if (this.mode === "edit" || this.mode === "create") {
        const v = Number(userId);
        if (!Number.isFinite(v) || v <= 0) {
          this.assigneeSearch.error = "Не удалось определить пользователя";
          return;
        }
        this.editor.form.assignee_id = String(v);
        this.selectedAssignee =
          (this.assigneeSearch.items || []).find((u) => u && u.id != null && Number(u.id) === v) || null;
        this.assigneeSearch.query = "";
        this.assigneeSearch.items = [];
        this.assigneeSearch.total = 0;
        this.assigneeSearch.error = "";
        return;
      }

      this.assigneeSave.error = "";
      if (this.assigneeSave.saving) return;

      const v = Number(userId);
      if (!Number.isFinite(v) || v <= 0) {
        this.assigneeSave.error = "Не удалось определить пользователя";
        return;
      }

      this.assigneeSave.saving = true;
      try {
        await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}`, {
          method: "PATCH",
          body: { assignee_id: v },
        });
        toastSuccess("Ответственный успешно назначен");
        await this.load();
        this.assigneeSearch.query = "";
        this.assigneeSearch.items = [];
        this.assigneeSearch.total = 0;
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось назначить ответственного");
      } finally {
        this.assigneeSave.saving = false;
      }
    },

    async clearAssignee() {
      if (this.mode === "edit" || this.mode === "create") {
        this.editor.form.assignee_id = "";
        this.selectedAssignee = null;
        return;
      }

      this.assigneeSave.error = "";
      if (this.assigneeSave.saving) return;

      this.assigneeSave.saving = true;
      try {
        await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}`, {
          method: "PATCH",
          body: { assignee_id: 0 },
        });
        toastSuccess("Ответственный успешно снят");
        await this.load();
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось снять ответственного");
      } finally {
        this.assigneeSave.saving = false;
      }
    },

    async assignToMe() {
      this.assigneeSave.error = "";
      if (this.assigneeSave.saving) return;

      const me = Alpine.store("auth").user || null;
      const id = me && me.id != null ? Number(me.id) : NaN;
      if (!Number.isFinite(id) || id <= 0) {
        this.assigneeSave.error = "Не удалось определить текущего пользователя";
        return;
      }

      if (this.mode === "edit" || this.mode === "create") {
        this.editor.form.assignee_id = String(id);
        this.selectedAssignee = me;
        this.assigneeSearch.query = "";
        this.assigneeSearch.items = [];
        this.assigneeSearch.total = 0;
        return;
      }

      await this.setAssignee(id);
    },

    onAttachmentFilesSelected(e) {
      this.attachmentUpload.error = "";
      const files = (e && e.target && e.target.files) ? Array.from(e.target.files) : [];
      this.attachmentUpload.files = files.filter((f) => f && f.size != null && f.size > 0);

      const safeFiles = Array.isArray(this.attachmentUpload.files) ? this.attachmentUpload.files : [];
      if (!safeFiles.length) {
        this.attachmentUpload.fileLabel = "Файл не выбран";
        return;
      }

      if (safeFiles.length === 1) {
        const name = safeFiles[0] && safeFiles[0].name ? String(safeFiles[0].name) : "";
        this.attachmentUpload.fileLabel = name || "Файл выбран";
        return;
      }

      const firstName = safeFiles[0] && safeFiles[0].name ? String(safeFiles[0].name) : "Файл";
      this.attachmentUpload.fileLabel = `${firstName} (+${safeFiles.length - 1})`;
    },

    formatBytes(bytes) {
      const n = Number(bytes);
      if (!Number.isFinite(n) || n <= 0) return "";
      const units = ["B", "KB", "MB", "GB"];
      let v = n;
      let i = 0;
      while (v >= 1024 && i < units.length - 1) {
        v /= 1024;
        i += 1;
      }
      const digits = i === 0 ? 0 : (v >= 10 ? 1 : 2);
      return `${v.toFixed(digits)} ${units[i]}`;
    },

    canManageAttachment(a) {
      const me = this._currentUser();
      if (!me) return false;
      if (me.role === "admin") return true;
      const uploaderId = a && a.uploaded_by && a.uploaded_by.id != null ? Number(a.uploaded_by.id) : NaN;
      const taskAuthorId = this.task && this.task.author && this.task.author.id != null ? Number(this.task.author.id) : NaN;
      if (Number.isFinite(uploaderId) && uploaderId > 0 && me.id === uploaderId) return true;
      return Number.isFinite(taskAuthorId) && taskAuthorId > 0 && me.id === taskAuthorId;
    },

    async uploadAttachments() {
      this.attachmentUpload.error = "";
      if (this.attachmentUpload.uploading) return;

      const files = Array.isArray(this.attachmentUpload.files) ? this.attachmentUpload.files : [];
      if (!files.length) {
        this.attachmentUpload.error = "Выберите файлы";
        return;
      }

      this.attachmentUpload.uploading = true;
      try {
        const failed = [];
        let okCount = 0;

        for (const file of files) {
          const fileName = file && file.name ? String(file.name) : "";
          const sizeBytes = file && file.size != null ? Number(file.size) : NaN;
          if (!fileName || !Number.isFinite(sizeBytes) || sizeBytes <= 0) continue;

          try {
            const { data: init } = await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}/attachments`, {
              method: "POST",
              body: { file_name: fileName, size_bytes: sizeBytes },
            });

            const uploadURL = init && init.upload_url ? String(init.upload_url) : "";
            const attachmentId = init && init.id != null ? init.id : null;
            if (!uploadURL || !attachmentId) throw new Error("Не удалось инициализировать загрузку");

            const putRes = await fetch(uploadURL, {
              method: "PUT",
              headers: { "Content-Type": file.type || "application/octet-stream" },
              body: file,
            });
            if (!putRes.ok) throw new Error(`Не удалось загрузить файл (${putRes.status})`);

            await window.apiFetch(
              `/tasks/${encodeURIComponent(this.taskId)}/attachments/${encodeURIComponent(attachmentId)}/confirm`,
              { method: "PUT" },
            );

            okCount += 1;
          } catch (err) {
            if (err && err.status === 401) return;

            const msg =
              (err && err.message ? String(err.message) : "") ||
              (err && err.status ? `HTTP ${err.status}` : "") ||
              "Не удалось загрузить файл";
            failed.push({ file, fileName, message: msg, err });
          }
        }

        if (okCount > 0) {
          await this.load();
        }

        if (!failed.length) {
          this.attachmentUpload.files = [];
          this.attachmentUpload.fileLabel = "Файл не выбран";
          toastSuccess(okCount > 1 ? "Файлы успешно загружены" : "Файл успешно загружен");
          return;
        }

        const failedFiles = failed.map((x) => x.file).filter(Boolean);
        this.attachmentUpload.files = failedFiles;
        if (!failedFiles.length) {
          this.attachmentUpload.fileLabel = "Файл не выбран";
        } else if (failedFiles.length === 1) {
          const name = failedFiles[0] && failedFiles[0].name ? String(failedFiles[0].name) : "";
          this.attachmentUpload.fileLabel = name || "Файл выбран";
        } else {
          const firstName = failedFiles[0] && failedFiles[0].name ? String(failedFiles[0].name) : "Файл";
          this.attachmentUpload.fileLabel = `${firstName} (+${failedFiles.length - 1})`;
        }

        this.attachmentUpload.error = failed
          .map((x) => `${x.fileName || "Файл"}: ${x.message || "Не удалось загрузить"}`)
          .join("\n");

        if (okCount > 0) {
          toastApiError(null, `Загружено: ${okCount}, ошибок: ${failed.length}`);
        } else {
          toastApiError(null, `Не удалось загрузить файлы (ошибок: ${failed.length})`);
        }
      } catch (err) {
        if (err && err.status === 401) return;
        if (err && err.status) {
          toastApiError(err, "Не удалось загрузить файлы");
        } else {
          this.attachmentUpload.error = err && err.message ? err.message : "Не удалось загрузить файлы";
        }
      } finally {
        this.attachmentUpload.uploading = false;
      }
    },

    async deleteAttachment(a) {
      this.openAttachmentDeleteModal(a);
    },

    openAttachmentDeleteModal(a) {
      this.attachmentDelete.error = "";
      this.attachmentDeleteModal.error = "";
      if (!a || a.id == null) {
        this.attachmentDelete.error = "Не удалось определить вложение";
        return;
      }
      if (!this.canManageAttachment(a)) return;

      this.attachmentDeleteModal.open = true;
      this.attachmentDeleteModal.attachmentId = a.id;
      this.attachmentDeleteModal.fileName = a.file_name ? String(a.file_name) : "";
      this.attachmentDeleteModal.saving = false;

      this.deletingAttachmentId = a.id;
    },

    closeAttachmentDeleteModal(force = false) {
      if (!force && this.attachmentDeleteModal.saving) return;
      this.attachmentDeleteModal.open = false;
      this.attachmentDeleteModal.attachmentId = null;
      this.attachmentDeleteModal.fileName = "";
      this.attachmentDeleteModal.saving = false;
      this.attachmentDeleteModal.error = "";
      this.deletingAttachmentId = null;
    },

    async confirmAttachmentDeleteModal() {
      this.attachmentDeleteModal.error = "";
      const id = this.attachmentDeleteModal.attachmentId;
      if (!id) {
        this.attachmentDeleteModal.error = "Не удалось определить вложение";
        return;
      }

      this.attachmentDeleteModal.saving = true;
      try {
        await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}/attachments/${encodeURIComponent(id)}`, {
          method: "DELETE",
        });
        toastSuccess("Вложение успешно удалено");
        this.closeAttachmentDeleteModal(true);
        await this.load();
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось удалить вложение");
      } finally {
        this.attachmentDeleteModal.saving = false;
      }
    },

    async submitComment() {
      this.commentForm.error = "";
      const content = this._markdownFieldValue("#task-new-comment", this.commentForm.content);
      if (!content) {
        this.commentForm.error = "Введите комментарий";
        return;
      }

      this.commentForm.saving = true;
      try {
        await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}/comments`, {
          method: "POST",
          body: { content },
        });
        this.commentForm.content = "";
        toastSuccess("Комментарий успешно добавлен");
        await this.load();
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось отправить комментарий");
      } finally {
        this.commentForm.saving = false;
      }
    },

    _currentUser() {
      const u = Alpine.store("auth").user || null;
      if (!u) return null;
      const id = u && u.id != null ? Number(u.id) : NaN;
      if (!Number.isFinite(id) || id <= 0) return null;
      const role = u && u.role ? String(u.role) : "";
      return { id, role };
    },

    canManageComment(c) {
      const me = this._currentUser();
      if (!me) return false;
      if (me.role === "admin") return true;
      const authorId = c && c.author && c.author.id != null ? Number(c.author.id) : NaN;
      return Number.isFinite(authorId) && authorId > 0 && me.id === authorId;
    },

    startEditComment(c) {
      if (!c || c.id == null) return;
      if (!this.canManageComment(c)) return;
      this.commentEdit.error = "";
      this.commentDelete.error = "";
      this.deletingCommentId = null;
      this.editingCommentId = c.id;
      this.editingContent = String(c.content || "");
      this.$nextTick(() => this.refreshMarkdownEditors());
    },

    cancelEditComment(silent = false) {
      this.editingCommentId = null;
      this.editingContent = "";
      this.commentEdit.saving = false;
      if (!silent) this.commentEdit.error = "";
      this.$nextTick(() => this.refreshMarkdownEditors());
    },

    openDeleteModal(c) {
      this.commentDelete.error = "";
      this.commentEdit.error = "";
      if (!c || c.id == null) return;
      if (!this.canManageComment(c)) return;

      this.deleteModal.open = true;
      this.deleteModal.commentId = c.id;
      this.deleteModal.preview = String(c.content || "").slice(0, 500);
      this.deleteModal.saving = false;
      this.deleteModal.error = "";

      this.deletingCommentId = c.id;
      this.cancelEditComment(true);
    },

    closeDeleteModal(force = false) {
      if (!force && this.deleteModal.saving) return;
      this.deleteModal.open = false;
      this.deleteModal.commentId = null;
      this.deleteModal.preview = "";
      this.deleteModal.saving = false;
      this.deleteModal.error = "";
      this.deletingCommentId = null;
    },

    async confirmDeleteModal() {
      this.deleteModal.error = "";
      const id = this.deleteModal.commentId;
      if (!id) {
        this.deleteModal.error = "Не удалось определить комментарий";
        return;
      }

      this.deleteModal.saving = true;
      try {
        await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}/comments/${encodeURIComponent(id)}`, {
          method: "DELETE",
        });
        toastSuccess("Комментарий успешно удалён");
        this.closeDeleteModal(true);
        await this.load();
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось удалить комментарий");
      } finally {
        this.deleteModal.saving = false;
      }
    },

    async saveEditComment() {
      this.commentEdit.error = "";
      const id = this.editingCommentId;
      if (!id) {
        this.commentEdit.error = "Не удалось определить комментарий";
        return;
      }

      const content = this._markdownFieldValue(
        `#task-comment-edit-${id}`,
        this.editingContent,
      );
      if (!content) {
        this.commentEdit.error = "Комментарий не должен быть пустым";
        return;
      }

      this.commentEdit.saving = true;
      try {
        await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}/comments/${encodeURIComponent(id)}`, {
          method: "PUT",
          body: { content },
        });
        toastSuccess("Комментарий успешно сохранён");
        await this.load();
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось сохранить комментарий");
      } finally {
        this.commentEdit.saving = false;
      }
    },

    async deleteComment(c) {
      this.openDeleteModal(c);
    },

    async changeStatus() {
      this.statusEdit.error = "";
      const status = (this.statusEdit.pending || "").trim();
      if (!status || status === (this.task && this.task.status || "")) {
        this.statusEdit.error = "Выберите другой статус";
        return;
      }

      this.statusEdit.saving = true;
      try {
        const body = { status };
        const comment = (this.statusEdit.comment || "").trim();
        if (comment) body.comment = comment;

        await window.apiFetch(`/tasks/${encodeURIComponent(this.taskId)}`, { method: "PATCH", body });
        toastSuccess("Статус задачи успешно изменён");
        this.statusEdit.comment = "";
        this.statusEdit.pending = "";
        await this.load();
      } catch (err) {
        if (err && err.status === 401) return;
        toastApiError(err, "Не удалось изменить статус");
      } finally {
        this.statusEdit.saving = false;
      }
    },
  };
};

window.taskPageFromLocation = function taskPageFromLocation() {
  try {
    let h = String(window.location.hash || "");
    if (h.startsWith("#")) h = h.slice(1);
    if (!h.startsWith("/")) h = `/${h}`;
    const parts = h.split("/").filter(Boolean);

    // Expected patterns:
    // - /tasks/new/:projectSlug?
    // - /tasks/:id
    // - /tasks/:id/edit
    const seg0 = (parts[0] || "").trim().toLowerCase();
    if (seg0 !== "tasks") return window.taskPage("", "view", "");

    const seg1 = (parts[1] || "").trim();
    if (seg1.toLowerCase() === "new") {
      const projectSlug = decodeURIComponent((parts[2] || "").trim());
      return window.taskPage("", "create", projectSlug);
    }

    const id = seg1;
    const seg2 = (parts[2] || "").trim().toLowerCase();
    const mode = seg2 === "edit" ? "edit" : "view";
    return window.taskPage(id, mode, "");
  } catch {
    return window.taskPage("", "view", "");
  }
};

(() => {
  const isSameOrigin = (url) => {
    try {
      const u = new URL(url, window.location.href);
      return u.origin === window.location.origin;
    } catch {
      return false;
    }
  };

  const PUBLIC_PATHS = new Set(["/login", "/register", "/pending"]);

  const isPublicPath = (path) => PUBLIC_PATHS.has(String(path || "/"));

  const getStoredToken = () => {
    try {
      return localStorage.getItem("access_token");
    } catch {
      return null;
    }
  };

  const pathWithoutQuery = (path) => String(path || "/").split("?")[0];

  const routeToPage = (path) => {
    const p = pathWithoutQuery(path);
    if (p === "/" || p === "") return "home.html";
    if (p === "/login") return "login.html";
    if (p === "/register") return "register.html";
    if (p === "/pending") return "pending.html";
    if (p === "/dashboards") return "dashboards.html";
    if (p === "/dashboards/personal") return "dashboards_personal.html";
    if (p === "/dashboards/tasks") return "dashboards_tasks.html";
    if (/^\/projects\/[^/]+\/tasks$/.test(p)) return "project_tasks.html";
    if (p === "/projects") return "projects.html";
    if (p === "/admin") return "admin.html";
    if (p === "/admin/users") return "admin_users.html";
    if (p === "/admin/projects") return "admin_projects.html";
    if (p === "/profile") return "profile.html";
    if (p.startsWith("/tasks/")) return "task.html";
    return "home.html";
  };

  const setTitleFromDoc = (doc) => {
    try {
      const t = doc && doc.querySelector ? doc.querySelector("title") : null;
      if (t && t.textContent) document.title = t.textContent;
    } catch {
      console.debug("SPA: failed to set title from loaded HTML");
    }
  };

  const initAlpineOn = (root) => {
    try {
      if (window.Alpine && typeof window.Alpine.initTree === "function") {
        window.Alpine.initTree(root);
      }
    } catch {
      console.debug("SPA: Alpine.initTree failed");
    }
  };

  const getHashPath = () => {
    let h = String(window.location.hash || "");
    if (h.startsWith("#")) h = h.slice(1);
    if (!h) return "/";
    if (!h.startsWith("/")) h = `/${h}`;
    return h;
  };

  let renderSeq = 0;

  const render = async () => {
    const seq = ++renderSeq;
    const root = document.getElementById("spa-root");
    if (!root) return;

    const path = getHashPath();
    const pathBase = pathWithoutQuery(path);
    const token = getStoredToken();

    if (!token && !isPublicPath(pathBase)) {
      window.location.hash = "/login";
      return;
    }

    if (token && (pathBase === "/login" || pathBase === "/register")) {
      window.location.hash = "/dashboards/personal";
      return;
    }

    if (token && (pathBase === "/" || pathBase === "")) {
      window.location.hash = "/dashboards/personal";
      return;
    }

    if (token && pathBase === "/dashboards") {
      window.location.hash = "/dashboards/personal";
      return;
    }

    if (token && pathBase === "/dashboards/tasks") {
      try {
        const qIdx = path.indexOf("?");
        const qs = new URLSearchParams(qIdx >= 0 ? path.slice(qIdx + 1) : "");
        const legacyProject = qs.get("project");
        if (legacyProject) {
          window.location.hash = `/projects/${encodeURIComponent(legacyProject.trim())}/tasks`;
          return;
        }
      } catch {
        console.debug("SPA: legacy project query redirect failed");
      }
    }

    try {
      if (window.Alpine && Alpine.store("nav")) {
        Alpine.store("nav").syncPath();
        Alpine.store("nav").trackPath(path);
      }
    } catch {
      console.debug("SPA: nav sync failed");
    }

    const page = routeToPage(pathBase);

    root.innerHTML = `<div class="text-sm text-slate-600">Загрузка…</div>`;

    try {
      const res = await fetch(`/static/pages/${page}`, { headers: { Accept: "text/html" } });
      if (!res.ok) throw new Error(`Failed to load page (${res.status})`);
      const html = await res.text();
      if (seq !== renderSeq) return;

      const doc = new DOMParser().parseFromString(html, "text/html");
      setTitleFromDoc(doc);
      const main = doc.querySelector("main");
      const content = main ? main.innerHTML : `<div class="text-sm text-red-600">Страница не содержит &lt;main&gt;.</div>`;
      if (seq !== renderSeq) return;
      root.innerHTML = content;
      initAlpineOn(root);
      initAlpineOn(document.body);
      try {
        if (window.Alpine && Alpine.store("nav")) Alpine.store("nav").syncPath();
      } catch {
        console.debug("SPA: nav sync after render failed");
      }
    } catch (err) {
      root.innerHTML = `<div class="rounded-lg border border-rose-200 bg-rose-50 p-3 text-sm text-rose-800">Не удалось загрузить страницу.</div>`;
      try {
        // eslint-disable-next-line no-console
        console.error(err);
      } catch {
        console.debug("SPA: failed to log render error");
      }
    }
  };

  const navigate = (to) => {
    const href = String(to || "/");
    if (href.startsWith("/api") || href.startsWith("/static") || href.startsWith("/pages")) {
      window.location.assign(href);
      return;
    }
    const hashIdx = href.indexOf("#");
    if (hashIdx >= 0) {
      const hashPart = href.slice(hashIdx + 1);
      window.location.hash = hashPart;
      return;
    }
    window.location.hash = href;
  };

  const shouldHandleClick = (a, e) => {
    if (!a || !a.getAttribute) return false;
    const hrefAttr = a.getAttribute("href");
    if (!hrefAttr) return false;
    if (hrefAttr.startsWith("#") && !hrefAttr.startsWith("#/")) return false;
    if (a.hasAttribute("download")) return false;
    if (a.getAttribute("target") && a.getAttribute("target") !== "_self") return false;
    if (e.defaultPrevented) return false;
    if (e.button !== 0) return false;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return false;
    if (!isSameOrigin(hrefAttr)) return false;
    return true;
  };

  window.__spaNavigate = navigate;

  window.addEventListener("hashchange", () => {
    try {
      if (window.Alpine && Alpine.store("nav")) Alpine.store("nav").syncPath();
    } catch {
      console.debug("SPA: nav sync on hashchange failed");
    }
    render();
  });
  document.addEventListener("click", (e) => {
    const a = e.target && e.target.closest ? e.target.closest("a") : null;
    if (!shouldHandleClick(a, e)) return;
    const href = a.getAttribute("href");
    if (!href) return;
    e.preventDefault();
    navigate(href);
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => render());
  } else {
    render();
  }
})();
