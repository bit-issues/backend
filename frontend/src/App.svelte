<script lang="ts">
  import Router from "$lib/router/router.svelte";
  import type { RouteDef } from "$lib/router/routes";
  import { navigate } from "$lib/router/routes";
  import { isAuthenticated, clear as clearAuth } from "$lib/stores/auth.svelte";
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

  let initialized = $state(false);

  let path = $state("");

  $effect(() => {
    setOnUnauthorized(() => {
      clearAuth();
      navigate("/login");
    });
    initialized = true;
  });

  $effect(() => {
    const handler = () => {
      path = window.location.hash.slice(1) || "/";
    };
    path = window.location.hash.slice(1) || "/";
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
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
