<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { getUser, isAdmin, logout } from "$lib/stores/auth.svelte";
  import { navigate } from "$lib/router/routes";
  import LayoutDashboardIcon from "@lucide/svelte/icons/layout-dashboard";
  import ListTodoIcon from "@lucide/svelte/icons/list-todo";
  import FolderKanbanIcon from "@lucide/svelte/icons/folder-kanban";
  import UsersIcon from "@lucide/svelte/icons/users";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import LogOutIcon from "@lucide/svelte/icons/log-out";
  import UserIcon from "@lucide/svelte/icons/user";
  import BugIcon from "@lucide/svelte/icons/bug";

  let { currentPath = "/" }: { currentPath?: string } = $props();

  let mobileOpen = $state(false);

  function isActive(pattern: string): boolean {
    if (pattern === "/") return currentPath === "/";
    return currentPath === pattern || currentPath.startsWith(`${pattern}/`);
  }

  async function handleLogout() {
    await logout();
    navigate("/login");
  }

  const mainNav = [
    { pattern: "/dashboard", label: "My Tasks", icon: LayoutDashboardIcon },
    { pattern: "/dashboard/all", label: "All Tasks", icon: ListTodoIcon },
    { pattern: "/projects", label: "Projects", icon: FolderKanbanIcon },
  ];

  const adminNav = [
    { pattern: "/admin/users", label: "Users", icon: UsersIcon },
    { pattern: "/admin/projects", label: "Projects", icon: SettingsIcon },
  ];
</script>

<!-- Mobile overlay -->
{#if mobileOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-40 bg-black/50 lg:hidden"
    onclick={() => (mobileOpen = false)}
  ></div>
{/if}

<!-- Mobile toggle -->
<button
  class="fixed top-3 left-3 z-50 flex size-9 items-center justify-center rounded-lg border border-border bg-background lg:hidden"
  onclick={() => (mobileOpen = !mobileOpen)}
  aria-label="Toggle navigation"
>
  <span class="text-foreground text-lg">{mobileOpen ? "✕" : "☰"}</span>
</button>

<!-- Sidebar -->
<aside
  class="border-border bg-card fixed top-0 left-0 z-50 flex h-full w-64 flex-col border-r pt-14 transition-transform lg:static lg:z-auto lg:pt-0 lg:translate-x-0 {mobileOpen
    ? 'translate-x-0'
    : '-translate-x-full'}"
>
  <!-- Logo -->
  <div class="flex items-center gap-2 border-b border-border px-5 py-4">
    <BugIcon class="size-5" />
    <span class="text-base font-semibold">BitIssues</span>
  </div>

  <!-- Navigation -->
  <nav class="flex-1 overflow-y-auto p-3">
    <div
      class="mb-2 px-2 text-xs font-medium uppercase tracking-wider text-muted-foreground"
    >
      Main
    </div>
    {#each mainNav as { pattern, label, icon: Icon }}
      <button
        type="button"
        onclick={() => {
          navigate(pattern);
          mobileOpen = false;
        }}
        class="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors {isActive(
          pattern,
        )
          ? 'bg-accent text-accent-foreground'
          : 'text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground'}"
      >
        <Icon class="size-4" />
        {label}
      </button>
    {/each}

    {#if isAdmin()}
      <div
        class="mt-4 mb-2 px-2 text-xs font-medium uppercase tracking-wider text-muted-foreground"
      >
        Admin
      </div>
      {#each adminNav as { pattern, label, icon: Icon }}
        <button
          type="button"
          onclick={() => {
            navigate(pattern);
            mobileOpen = false;
          }}
          class="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors {isActive(
            pattern,
          )
            ? 'bg-accent text-accent-foreground'
            : 'text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground'}"
        >
          <Icon class="size-4" />
          {label}
        </button>
      {/each}
    {/if}
  </nav>

  <!-- User section -->
  <div class="border-t border-border p-3">
    <div class="mb-2 flex items-center gap-2 px-2 py-1">
      <div
        class="bg-muted flex size-7 items-center justify-center rounded-full text-xs font-medium"
      >
        {(getUser()?.name?.[0] || "?").toUpperCase()}
      </div>
      <div class="min-w-0 flex-1">
        <p class="truncate text-sm font-medium">{getUser()?.name || "User"}</p>
        <p class="truncate text-xs text-muted-foreground">
          {getUser()?.email || ""}
        </p>
      </div>
    </div>
    <div class="flex gap-1">
      <Button
        variant="ghost"
        size="sm"
        class="flex-1"
        onclick={() => {
          navigate("/profile");
          mobileOpen = false;
        }}
      >
        <UserIcon class="size-4" />
        Profile
      </Button>
      <Button variant="ghost" size="sm" class="flex-1" onclick={handleLogout}>
        <LogOutIcon class="size-4" />
        Logout
      </Button>
    </div>
  </div>
</aside>
