<script lang="ts">
  import { onMount } from "svelte";
  import { listProjects } from "$lib/api/projects";
  import { navigate } from "$lib/router/routes";
  import {
    enrichRecentFromProjects,
    touchProject,
  } from "$lib/stores/recent-projects.svelte";
  import * as Card from "$lib/components/ui/card";
  import { Button } from "$lib/components/ui/button";
  import FolderKanbanIcon from "@lucide/svelte/icons/folder-kanban";
  import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
  import SearchIcon from "@lucide/svelte/icons/search";
  import XIcon from "@lucide/svelte/icons/x";
  import type { Project } from "$lib/types/api";
  import {
    parseFilters,
    replaceFilters,
    type FilterState,
  } from "$lib/router/filter-url";
  import { createLatestRequestGuard, runLatest } from "$lib/latest-request";

  let { params = {} }: { params?: Record<string, string> } = $props();

  const FILTER_DEFAULTS: FilterState = {
    project: "",
    statuses: [],
    priorities: [],
    search: "",
    sort: "-created_at",
    offset: 0,
  };

  let projects = $state<Project[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state("");
  const limit = 20;
  const initial = parseFilters(window.location.hash, FILTER_DEFAULTS);
  let offset = $state(initial.offset);
  let searchTerm = $state(initial.search);
  let debouncedSearch = $state(initial.search);
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;
  const requestGuard = createLatestRequestGuard();

  function handleSearchInput(value: string) {
    searchTerm = value;
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      debouncedSearch = value;
      offset = 0; // Reset pagination when search changes
    }, 300);
  }

  function clearSearch() {
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = null;
    searchTerm = "";
    debouncedSearch = "";
    offset = 0;
  }

  function load() {
    loading = true;
    error = "";
    runLatest(
      requestGuard,
      () => listProjects(limit, offset, debouncedSearch || undefined),
      {
        onSuccess: (res) => {
          projects = res.items;
          total = res.total;
          const pageOffset = Math.floor(offset / limit) * limit;
          const lastOffset =
            total > 0 ? Math.floor((total - 1) / limit) * limit : 0;
          const nextOffset = Math.min(pageOffset, lastOffset);
          if (offset !== nextOffset) {
            offset = nextOffset;
          }
          enrichRecentFromProjects(res.items);
        },
        onError: (e) => {
          error = (e as Error).message || "Failed to load projects";
        },
        onFinally: () => {
          loading = false;
        },
      },
    );
  }

  $effect(() => {
    replaceFilters(
      {
        project: "",
        statuses: [],
        priorities: [],
        search: debouncedSearch,
        sort: FILTER_DEFAULTS.sort,
        offset,
      },
      FILTER_DEFAULTS,
    );
  });

  $effect(() => {
    const handler = () => {
      if (searchTimeout) clearTimeout(searchTimeout);
      searchTimeout = null;
      const next = parseFilters(window.location.hash, FILTER_DEFAULTS);
      if (searchTerm !== next.search) searchTerm = next.search;
      if (debouncedSearch !== next.search) debouncedSearch = next.search;
      if (offset !== next.offset) offset = next.offset;
    };
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  });

  $effect(load);
</script>

<div class="mx-auto max-w-4xl space-y-6">
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-semibold">Projects</h1>
      <p class="text-muted-foreground text-sm">
        {total} project{total !== 1 ? "s" : ""}
      </p>
    </div>
  </div>

  <div class="relative">
    <SearchIcon
      class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
    />
    <input
      type="text"
      placeholder="Search projects by name or slug..."
      value={searchTerm}
      oninput={(e) => handleSearchInput((e.target as HTMLInputElement).value)}
      class="w-full rounded-md border border-input bg-background py-2 pl-10 pr-10 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
    />
    {#if searchTerm}
      <button
        onclick={clearSearch}
        class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
        aria-label="Clear search"
      >
        <XIcon class="size-4" />
      </button>
    {/if}
  </div>

  {#if loading}
    <p class="text-muted-foreground py-8 text-center">Loading...</p>
  {:else if error}
    <p class="text-destructive py-8 text-center">{error}</p>
  {:else if projects.length === 0}
    <Card.Root class="py-12">
      <Card.CardContent class="flex flex-col items-center gap-2">
        {#if debouncedSearch}
          <SearchIcon class="text-muted-foreground size-12" />
          <p class="text-muted-foreground text-sm">
            No projects found matching "{debouncedSearch}"
          </p>
        {:else}
          <FolderKanbanIcon class="text-muted-foreground size-12" />
          <p class="text-muted-foreground text-sm">No projects yet</p>
        {/if}
      </Card.CardContent>
    </Card.Root>
  {:else}
    <div class="grid gap-4 sm:grid-cols-2">
      {#each projects as project (project.id)}
        <Card.Root
          class="cursor-pointer"
          onclick={() => {
            touchProject({ id: project.id, name: project.name });
            navigate(`/projects/${project.id}`);
          }}
        >
          <Card.CardHeader>
            <Card.CardTitle>{project.name}</Card.CardTitle>
            <Card.CardDescription class="flex items-center gap-1">
              <span class="truncate">{project.repo_url}</span>
              <ExternalLinkIcon class="size-3 shrink-0" />
            </Card.CardDescription>
          </Card.CardHeader>
          <Card.CardContent>
            <p class="text-muted-foreground text-xs">
              Created {new Date(project.created_at).toLocaleDateString()}
            </p>
          </Card.CardContent>
        </Card.Root>
      {/each}
    </div>

    {#if total > limit}
      <div class="flex items-center justify-center gap-2">
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
          Page {Math.floor(offset / limit) + 1} of {Math.ceil(total / limit)}
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
