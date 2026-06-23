<script lang="ts">
  import Router from "$lib/router/router.svelte";
  import type { RouteDef } from "$lib/router/routes";
  import { navigate } from "$lib/router/routes";
  import { isAuthenticated } from "$lib/stores/auth.svelte";
  import { trackRecentFromPath } from "$lib/stores/recent-projects.svelte";
  import { setOnUnauthorized } from "$lib/api/client";
  import AppShell from "$lib/components/AppShell.svelte";
  import { Toaster } from "$lib/components/ui/sonner";

  import HomePage from "./pages/home.svelte";
  import LoginPage from "./pages/login.svelte";
  import RegisterPage from "./pages/register.svelte";
  import PendingPage from "./pages/pending.svelte";
  import NotFoundPage from "./pages/notfound.svelte";

  import DashboardPersonal from "$lib/pages/dashboard-personal.svelte";
  import DashboardTasks from "$lib/pages/dashboard-tasks.svelte";
  import ProjectsPage from "$lib/pages/projects.svelte";
  import ProjectTasks from "$lib/pages/project-tasks.svelte";
  import TaskDetailPage from "$lib/pages/task-detail.svelte";
  import TaskNewPage from "$lib/pages/task-new.svelte";
  import TaskEditPage from "$lib/pages/task-edit.svelte";

  import AdminPage from "$lib/pages/admin.svelte";
  import AdminUsers from "$lib/pages/admin-users.svelte";
  import AdminProjects from "$lib/pages/admin-projects.svelte";
  import ProfilePage from "$lib/pages/profile.svelte";
  import SecurityPage from "$lib/pages/settings/security.svelte";

  let initialized = $state(false);

  let path = $state("");

  $effect(() => {
    setOnUnauthorized(() => {
      navigate("/login");
    });
    initialized = true;
  });

  $effect(() => {
    const handler = () => {
      path = (window.location.hash.slice(1) || "/").split("?")[0];
    };
    path = (window.location.hash.slice(1) || "/").split("?")[0];
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  });

  $effect(() => {
    const syncRecent = () => {
      trackRecentFromPath(window.location.hash.slice(1) || "/");
    };
    syncRecent();
    window.addEventListener("hashchange", syncRecent);
    return () => window.removeEventListener("hashchange", syncRecent);
  });

  const routes: RouteDef[] = [
    { pattern: "/", component: HomePage, auth: true },
    { pattern: "/login", component: LoginPage },
    { pattern: "/register", component: RegisterPage },
    { pattern: "/pending", component: PendingPage },
    { pattern: "/dashboard", component: DashboardPersonal, auth: true },
    { pattern: "/dashboard/all", component: DashboardTasks, auth: true },
    { pattern: "/projects", component: ProjectsPage, auth: true },
    { pattern: "/projects/:slug", component: ProjectTasks, auth: true },
    { pattern: "/tasks/new", component: TaskNewPage, auth: true },
    { pattern: "/tasks/:id/edit", component: TaskEditPage, auth: true },
    { pattern: "/tasks/:id", component: TaskDetailPage, auth: true },
    { pattern: "/profile", component: ProfilePage, auth: true },
    { pattern: "/settings/security", component: SecurityPage, auth: true },
    { pattern: "/admin/users", component: AdminUsers, auth: true, role: "admin" },
    { pattern: "/admin/projects", component: AdminProjects, auth: true, role: "admin" },
    { pattern: "/admin", component: AdminPage, auth: true, role: "admin" },
  ];
</script>

<Toaster />

{#if initialized}
  {#if isAuthenticated()}
    <AppShell currentPath={path}>
      <Router {routes} notFound={NotFoundPage} />
    </AppShell>
  {:else}
    <Router {routes} notFound={NotFoundPage} />
  {/if}
{:else}
  <div class="flex min-h-screen items-center justify-center">
    <p class="text-muted-foreground text-sm">Loading...</p>
  </div>
{/if}
