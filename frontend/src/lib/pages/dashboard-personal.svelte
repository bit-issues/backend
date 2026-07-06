<script lang="ts">
  import { getMyTasks } from "$lib/api/tasks";
  import { listProjects } from "$lib/api/projects";
  import { navigate } from "$lib/router/routes";
  import { getUser } from "$lib/stores/auth.svelte";
  import TaskTable from "$lib/components/TaskTable.svelte";
  import TaskFilters from "$lib/components/TaskFilters.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import type { Task, Project } from "$lib/types/api";
  import { ACTIVE_STATUSES } from "$lib/types/api";
  import { getSnapshot, saveSnapshot } from "$lib/stores/task-filters.svelte";

  let { params = {} }: { params?: Record<string, string> } = $props();

  const ROUTE_KEY = "/dashboard";

  let createdTasks = $state<Task[]>([]);
  let assignedTasks = $state<Task[]>([]);
  let projects = $state<Project[]>([]);
  let loading = $state(true);
  let error = $state("");
  let activeTab = $state<"created" | "assigned">("assigned");

  let filterProject = $state("");
  let filterStatuses = $state<string[]>([...ACTIVE_STATUSES]);
  let filterPriorities = $state<string[]>([]);
  let searchQuery = $state("");
  let sort = $state("-created_at");

  let saved = getSnapshot(ROUTE_KEY);
  if (saved) {
    filterProject = saved.filterProject ?? "";
    filterStatuses = saved.filterStatuses;
    filterPriorities = saved.filterPriorities;
    sort = saved.sort;
  }

  function toggleStatus(s: string) {
    if (filterStatuses.includes(s)) {
      filterStatuses = filterStatuses.filter((x) => x !== s);
    } else {
      filterStatuses = [...filterStatuses, s];
    }
  }

  function togglePriority(p: string) {
    if (filterPriorities.includes(p)) {
      filterPriorities = filterPriorities.filter((x) => x !== p);
    } else {
      filterPriorities = [...filterPriorities, p];
    }
  }

  function handleProjectChange(v: string) {
    filterProject = v;
  }

  function resetFilters() {
    filterProject = "";
    filterStatuses = [...ACTIVE_STATUSES];
    filterPriorities = [];
    searchQuery = "";
  }

  function handleSort(field: string) {
    sort = sort === field ? `-${field}` : field;
  }

  function handleTaskClick(task: Task) {
    navigate(`/tasks/${task.project_slug}/${task.number}`);
  }

  $effect(() => {
    saveSnapshot(ROUTE_KEY, {
      filterProject: filterProject || undefined,
      filterStatuses: [...filterStatuses],
      filterPriorities: [...filterPriorities],
      sort,
      offset: 0,
    });
  });

  $effect(() => {
    loading = true;
    error = "";

    const filters: Record<string, string | number | undefined> = {
      sort,
    };
    if (filterProject) filters.project = filterProject;
    if (filterStatuses.length) filters.statuses = filterStatuses.join(",");
    if (filterPriorities.length)
      filters.priorities = filterPriorities.join(",");
    if (searchQuery) filters.search = searchQuery;

    Promise.all([
      listProjects(100, 0).then((r) => {
        projects = r.items;
      }),
      getMyTasks({ limit: 50, ...filters }).then((res) => {
        const currentUserId = getUser()?.id;
        createdTasks = res.items.filter((t) => t.author.id === currentUserId);
        assignedTasks = res.items.filter(
          (t) => t.assignee?.id === currentUserId,
        );
      }),
    ])
      .catch((e) => {
        error = e.message || "Failed to load tasks";
      })
      .finally(() => {
        loading = false;
      });
  });
</script>

<div class="mx-auto max-w-7xl space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-semibold">My Dashboard</h1>
      <p class="text-muted-foreground text-sm">
        Tasks assigned to or created by you
      </p>
    </div>
    <Button onclick={() => navigate("/tasks/new")}>New task</Button>
  </div>

  <div class="grid grid-cols-[1fr_280px] gap-6">
    <div class="min-w-0">
      {#if loading}
        <p class="text-muted-foreground py-8 text-center">Loading...</p>
      {:else if error}
        <p class="text-destructive py-8 text-center">{error}</p>
      {:else}
        <div class="flex border-b mb-4">
          <button
            type="button"
            onclick={() => (activeTab = "assigned")}
            class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors {activeTab ===
            'assigned'
              ? 'border-primary text-foreground'
              : 'border-transparent text-muted-foreground hover:text-foreground'}"
          >
            Assigned to me ({assignedTasks.length})
          </button>
          <button
            type="button"
            onclick={() => (activeTab = "created")}
            class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors {activeTab ===
            'created'
              ? 'border-primary text-foreground'
              : 'border-transparent text-muted-foreground hover:text-foreground'}"
          >
            Created by me ({createdTasks.length})
          </button>
        </div>

        {#if activeTab === "assigned"}
          <Card.Root>
            <Card.CardHeader>
              <Card.CardTitle>Assigned to me</Card.CardTitle>
              <Card.CardDescription
                >{assignedTasks.length} task{assignedTasks.length !== 1
                  ? "s"
                  : ""}</Card.CardDescription
              >
            </Card.CardHeader>
            <Card.CardContent>
              <TaskTable
                tasks={assignedTasks}
                {sort}
                onSort={handleSort}
                onTaskClick={handleTaskClick}
              />
            </Card.CardContent>
          </Card.Root>
        {:else}
          <Card.Root>
            <Card.CardHeader>
              <Card.CardTitle>Created by me</Card.CardTitle>
              <Card.CardDescription
                >{createdTasks.length} task{createdTasks.length !== 1
                  ? "s"
                  : ""}</Card.CardDescription
              >
            </Card.CardHeader>
            <Card.CardContent>
              <TaskTable
                tasks={createdTasks}
                {sort}
                onSort={handleSort}
                onTaskClick={handleTaskClick}
              />
            </Card.CardContent>
          </Card.Root>
        {/if}
      {/if}
    </div>

    <aside>
      <TaskFilters
        {projects}
        {searchQuery}
        {filterProject}
        {filterStatuses}
        {filterPriorities}
        onSearchChange={(v) => {
          searchQuery = v;
        }}
        onProjectChange={handleProjectChange}
        onStatusToggle={toggleStatus}
        onPriorityToggle={togglePriority}
        onReset={resetFilters}
      />
    </aside>
  </div>
</div>
