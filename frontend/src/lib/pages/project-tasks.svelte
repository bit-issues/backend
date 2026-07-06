<script lang="ts">
  import { untrack } from "svelte";
  import { getProject } from "$lib/api/projects";
  import {
    listTasks,
    type TaskFilters as TaskFilterParams,
  } from "$lib/api/tasks";
  import { navigate } from "$lib/router/routes";
  import { touchProject } from "$lib/stores/recent-projects.svelte";
  import TaskTable from "$lib/components/TaskTable.svelte";
  import TaskFilters from "$lib/components/TaskFilters.svelte";
  import * as Card from "$lib/components/ui/card";
  import { Button } from "$lib/components/ui/button";
  import type { Project, Task } from "$lib/types/api";
  import { ACTIVE_STATUSES } from "$lib/types/api";
  import { getSnapshot, saveSnapshot } from "$lib/stores/task-filters.svelte";

  // svelte-ignore a11y_click_events_have_key_events

  let { params = {} }: { params?: Record<string, string> } = $props();
  let slug = $derived(params.slug || "");
  let routeKey = $derived(`/projects/${slug}`);

  let project = $state<Project | null>(null);
  let tasks = $state<Task[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state("");
  let currentRouteKey = $state("");
  let sort = $state("-created_at");
  let offset = $state(0);
  const limit = 50;

  let filterStatuses = $state<string[]>([...ACTIVE_STATUSES]);
  let filterPriorities = $state<string[]>([]);
  let searchQuery = $state("");

  $effect(() => {
    const key = routeKey;
    const saved = untrack(() => getSnapshot(key));
    if (saved) {
      filterStatuses = saved.filterStatuses;
      filterPriorities = saved.filterPriorities;
      sort = saved.sort;
      offset = saved.offset;
    } else {
      searchQuery = "";
      sort = "-created_at";
    }
  });

  function toggleStatus(s: string) {
    if (filterStatuses.includes(s)) {
      filterStatuses = filterStatuses.filter((x) => x !== s);
    } else {
      filterStatuses = [...filterStatuses, s];
    }
    offset = 0;
  }

  function togglePriority(p: string) {
    if (filterPriorities.includes(p)) {
      filterPriorities = filterPriorities.filter((x) => x !== p);
    } else {
      filterPriorities = [...filterPriorities, p];
    }
    offset = 0;
  }

  function resetFilters() {
    filterStatuses = [...ACTIVE_STATUSES];
    filterPriorities = [];
    searchQuery = "";
    offset = 0;
  }

  function load() {
    if (!slug) return;
    loading = true;
    error = "";

    const filters: TaskFilterParams = { project: slug, sort, limit, offset };
    if (filterStatuses.length) filters.statuses = filterStatuses.join(",");
    if (filterPriorities.length)
      filters.priorities = filterPriorities.join(",");
    if (searchQuery) filters.search = searchQuery;

    Promise.all([getProject(slug), listTasks(filters)])
      .then(([proj, res]) => {
        project = proj;
        tasks = res.items;
        total = res.total;
        touchProject({ id: proj.id, name: proj.name });
      })
      .catch((e) => {
        error = e.message || "Failed to load data";
      })
      .finally(() => {
        currentRouteKey = routeKey;
        loading = false;
      });
  }

  $effect(load);

  $effect(() => {
    if (routeKey !== currentRouteKey) return;
    saveSnapshot(routeKey, {
      filterStatuses: [...filterStatuses],
      filterPriorities: [...filterPriorities],
      sort,
      offset,
    });
  });

  function handleSort(field: string) {
    sort = sort === field ? `-${field}` : field;
    offset = 0;
  }

  function handleTaskClick(task: Task) {
    navigate(`/tasks/${task.project_slug}/${task.number}`);
  }
</script>

<div class="mx-auto max-w-7xl space-y-4">
  {#if loading}
    <p class="text-muted-foreground py-8 text-center">Loading...</p>
  {:else if error}
    <p class="text-destructive py-8 text-center">{error}</p>
  {:else if project}
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold">{project.name}</h1>
        <p class="text-muted-foreground text-sm">
          {total} task{total !== 1 ? "s" : ""}
        </p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" onclick={() => navigate("/projects")}>
          All projects
        </Button>
        <Button onclick={() => navigate(`/tasks/new?project=${slug}`)}>
          New task
        </Button>
      </div>
    </div>

    <div class="grid grid-cols-[1fr_280px] gap-6">
      <div class="min-w-0">
        <Card.Root>
          <Card.CardContent class="p-0">
            <TaskTable
              {tasks}
              {sort}
              onSort={handleSort}
              onTaskClick={handleTaskClick}
            />
          </Card.CardContent>
        </Card.Root>

        {#if total > limit}
          <div class="flex items-center justify-center gap-2 pt-4">
            <Button
              variant="outline"
              size="sm"
              disabled={offset === 0}
              onclick={() => {
                offset = Math.max(0, offset - limit);
                load();
              }}
            >
              Previous
            </Button>
            <span class="text-muted-foreground text-sm">
              Page {Math.floor(offset / limit) + 1} of {Math.ceil(
                total / limit,
              )}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={offset + limit >= total}
              onclick={() => {
                offset = offset + limit;
                load();
              }}
            >
              Next
            </Button>
          </div>
        {/if}
      </div>

      <aside>
        <TaskFilters
          {searchQuery}
          {filterStatuses}
          {filterPriorities}
          hideProject
          onSearchChange={(v) => {
            searchQuery = v;
            offset = 0;
          }}
          onStatusToggle={toggleStatus}
          onPriorityToggle={togglePriority}
          onReset={resetFilters}
        />
      </aside>
    </div>
  {/if}
</div>
