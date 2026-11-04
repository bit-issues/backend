<script lang="ts">
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
  import {
    parseFilters,
    replaceFilters,
    type FilterState,
  } from "$lib/router/filter-url";
  import { createLatestRequestGuard, runLatest } from "$lib/latest-request";

  // svelte-ignore a11y_click_events_have_key_events

  let { params = {} }: { params?: Record<string, string> } = $props();
  let slug = $derived(params.slug || "");

  let project = $state<Project | null>(null);
  let tasks = $state<Task[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state("");
  const limit = 50;
  const requestGuard = createLatestRequestGuard();

  const FILTER_DEFAULTS: FilterState = {
    project: "",
    statuses: [...ACTIVE_STATUSES],
    priorities: [],
    search: "",
    sort: "-created_at",
    offset: 0,
  };

  const initial = parseFilters(window.location.hash, FILTER_DEFAULTS);
  let filterStatuses = $state(initial.statuses);
  let filterPriorities = $state(initial.priorities);
  let searchQuery = $state(initial.search);
  let sort = $state(initial.sort);
  let offset = $state(initial.offset);
  let lastSlug = "";

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
    if (slug !== lastSlug) {
      lastSlug = slug;
      project = null;
    }
    loading = true;
    error = "";

    const filters: TaskFilterParams = { project: slug, sort, limit, offset };
    if (filterStatuses.length) filters.statuses = filterStatuses.join(",");
    if (filterPriorities.length)
      filters.priorities = filterPriorities.join(",");
    if (searchQuery) filters.search = searchQuery;

    runLatest(
      requestGuard,
      () => Promise.all([getProject(slug), listTasks(filters)]),
      {
        onSuccess: ([proj, res]) => {
          project = proj;
          tasks = res.items;
          total = res.total;
          const pageOffset = Math.floor(offset / limit) * limit;
          const lastOffset =
            total > 0 ? Math.floor((total - 1) / limit) * limit : 0;
          const nextOffset = Math.min(pageOffset, lastOffset);
          if (offset !== nextOffset) {
            offset = nextOffset;
          }
          touchProject({ id: proj.id, name: proj.name });
        },
        onError: (e) => {
          error = (e as Error).message || "Failed to load data";
        },
        onFinally: () => {
          loading = false;
        },
      },
    );
  }

  $effect(load);

  $effect(() => {
    replaceFilters(
      {
        project: "",
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

  function handleSort(field: string) {
    sort = sort === field ? `-${field}` : field;
    offset = 0;
  }

  function handleTaskClick(task: Task) {
    navigate(`/tasks/${task.project_slug}/${task.number}`);
  }

  let initialLoading = $derived(loading && project === null);
</script>

<div class="mx-auto max-w-7xl space-y-4">
  {#if initialLoading}
    <p class="text-muted-foreground py-8 text-center">Loading...</p>
  {:else if error && !project}
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
        {#if loading}
          <p class="text-muted-foreground py-2 text-center text-sm">
            Loading...
          </p>
        {/if}
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
