<script lang="ts">
  import { listProjects } from "$lib/api/projects";
  import {
    listTasks,
    type TaskFilters as TaskFilterParams,
  } from "$lib/api/tasks";
  import { navigate } from "$lib/router/routes";
  import TaskTable from "$lib/components/TaskTable.svelte";
  import * as Card from "$lib/components/ui/card";
  import TaskFilters from "$lib/components/TaskFilters.svelte";
  import { Button } from "$lib/components/ui/button";
  import type { Project, Task } from "$lib/types/api";
  import { ACTIVE_STATUSES } from "$lib/types/api";
  import {
    parseFilters,
    replaceFilters,
    type FilterState,
  } from "$lib/router/filter-url";
  import { createLatestRequestGuard, runLatest } from "$lib/latest-request";

  let { params = {} }: { params?: Record<string, string> } = $props();

  const FILTER_DEFAULTS: FilterState = {
    project: "",
    statuses: [...ACTIVE_STATUSES],
    priorities: [],
    search: "",
    sort: "-created_at",
    offset: 0,
  };

  let tasks = $state<Task[]>([]);
  let total = $state(0);
  let projects = $state<Project[]>([]);
  let loading = $state(true);
  let error = $state("");

  const initial = parseFilters(window.location.hash, FILTER_DEFAULTS);
  let filterProject = $state(initial.project);
  let filterStatuses = $state(initial.statuses);
  let filterPriorities = $state(initial.priorities);
  let searchQuery = $state(initial.search);
  let sort = $state(initial.sort);
  let offset = $state(initial.offset);
  const limit = 50;

  const requestGuard = createLatestRequestGuard();

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

  function handleProjectChange(v: string) {
    filterProject = v;
    offset = 0;
  }

  function handleSort(field: string) {
    sort = sort === field ? `-${field}` : field;
    offset = 0;
  }

  function load() {
    loading = true;
    error = "";

    const filters: TaskFilterParams = {
      sort,
      limit,
      offset,
    };
    if (filterProject) filters.project = filterProject;
    if (filterStatuses.length) filters.statuses = filterStatuses.join(",");
    if (filterPriorities.length)
      filters.priorities = filterPriorities.join(",");
    if (searchQuery) filters.search = searchQuery;

    runLatest(
      requestGuard,
      () =>
        Promise.all([
          listProjects(100, 0).then((r) => r.items),
          listTasks(filters).then((r) => ({ items: r.items, total: r.total })),
        ]),
      {
        onSuccess: ([projectItems, taskRes]) => {
          projects = projectItems;
          tasks = taskRes.items;
          total = taskRes.total;
          const pageOffset = Math.floor(offset / limit) * limit;
          const lastOffset =
            total > 0 ? Math.floor((total - 1) / limit) * limit : 0;
          const nextOffset = Math.min(pageOffset, lastOffset);
          if (offset !== nextOffset) {
            offset = nextOffset;
          }
        },
        onError: (e) => {
          error = (e as Error).message || "Failed to load tasks";
        },
        onFinally: () => {
          loading = false;
        },
      },
    );
  }

  function resetFilters() {
    filterProject = "";
    filterStatuses = [...ACTIVE_STATUSES];
    filterPriorities = [];
    searchQuery = "";
    offset = 0;
  }

  $effect(load);

  $effect(() => {
    replaceFilters(
      {
        project: filterProject,
        statuses: [...filterStatuses],
        priorities: [...filterPriorities],
        search: searchQuery,
        sort,
        offset,
      },
      FILTER_DEFAULTS,
    );
  });

  $effect(() => {
    const handler = () => {
      const next = parseFilters(window.location.hash, FILTER_DEFAULTS);
      if (filterProject !== next.project) filterProject = next.project;
      if (filterStatuses.join(",") !== next.statuses.join(",")) {
        filterStatuses = next.statuses;
      }
      if (filterPriorities.join(",") !== next.priorities.join(",")) {
        filterPriorities = next.priorities;
      }
      if (searchQuery !== next.search) searchQuery = next.search;
      if (sort !== next.sort) sort = next.sort;
      if (offset !== next.offset) offset = next.offset;
    };
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  });

  function handleTaskClick(task: Task) {
    navigate(`/tasks/${task.project_slug}/${task.number}`);
  }
</script>

<div class="mx-auto max-w-7xl space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-semibold">All Tasks</h1>
      <p class="text-muted-foreground text-sm">
        {total} task{total !== 1 ? "s" : ""}
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
          offset = 0;
        }}
        onProjectChange={handleProjectChange}
        onStatusToggle={toggleStatus}
        onPriorityToggle={togglePriority}
        onReset={resetFilters}
      />
    </aside>
  </div>
</div>
