<script lang="ts">
  import { untrack } from "svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import {
    STATUSES,
    PRIORITIES,
    ACTIVE_STATUSES,
    type Project,
  } from "$lib/types/api";

  let {
    projects = [],
    searchQuery = "",
    filterProject = "",
    filterStatuses = [],
    filterPriorities = [],
    hideProject = false,
    onSearchChange,
    onProjectChange,
    onStatusToggle,
    onPriorityToggle,
    onReset,
  }: {
    projects?: Project[];
    searchQuery?: string;
    filterProject?: string;
    filterStatuses?: string[];
    filterPriorities?: string[];
    hideProject?: boolean;
    onSearchChange?: (v: string) => void;
    onProjectChange?: (v: string) => void;
    onStatusToggle?: (s: string) => void;
    onPriorityToggle?: (p: string) => void;
    onReset?: () => void;
  } = $props();

  // Local draft for instant input feedback
  let draft = $state(untrack(() => searchQuery));

  // Sync from parent when searchQuery changes externally (e.g., reset)
  $effect(() => {
    draft = searchQuery;
  });

  let mounted = $state(false);
  $effect(() => {
    mounted = true;
  });

  // Debounce parent notification
  $effect(() => {
    if (!mounted) return;
    const value = draft;
    const timer = setTimeout(() => {
      untrack(() => onSearchChange?.(value));
    }, 250);
    return () => clearTimeout(timer);
  });

  const statusFilterColors: Record<string, string> = {
    New: "bg-blue-700/10 text-blue-700 ring-1 ring-inset ring-blue-200 dark:bg-blue-300/15 dark:text-blue-300 dark:ring-blue-700",
    Open: "bg-sky-700/10 text-sky-700 ring-1 ring-inset ring-sky-200 dark:bg-sky-300/15 dark:text-sky-300 dark:ring-sky-700",
    "In Progress":
      "bg-amber-700/10 text-amber-700 ring-1 ring-inset ring-amber-200 dark:bg-amber-300/15 dark:text-amber-300 dark:ring-amber-700",
    Resolved:
      "bg-emerald-700/10 text-emerald-700 ring-1 ring-inset ring-emerald-200 dark:bg-emerald-300/15 dark:text-emerald-300 dark:ring-emerald-700",
    Closed:
      "bg-slate-500/10 text-slate-500 ring-1 ring-inset ring-slate-200 dark:bg-slate-400/15 dark:text-slate-400 dark:ring-slate-600",
    Reopened:
      "bg-red-700/10 text-red-700 ring-1 ring-inset ring-red-200 dark:bg-red-300/15 dark:text-red-300 dark:ring-red-700",
    Invalid:
      "bg-gray-400/10 text-gray-400 ring-1 ring-inset ring-gray-200 dark:bg-gray-500/15 dark:text-gray-500 dark:ring-gray-600",
    Duplicate:
      "bg-gray-400/10 text-gray-400 ring-1 ring-inset ring-gray-200 dark:bg-gray-500/15 dark:text-gray-500 dark:ring-gray-600",
    Wontfix:
      "bg-gray-400/10 text-gray-400 ring-1 ring-inset ring-gray-200 dark:bg-gray-500/15 dark:text-gray-500 dark:ring-gray-600",
    "On Hold":
      "bg-orange-700/10 text-orange-700 ring-1 ring-inset ring-orange-200 dark:bg-orange-300/15 dark:text-orange-300 dark:ring-orange-700",
  };

  const priorityFilterColors: Record<string, string> = {
    Blocker:
      "bg-red-700/10 text-red-700 ring-1 ring-inset ring-red-200 dark:bg-red-300/15 dark:text-red-300 dark:ring-red-700",
    Critical:
      "bg-orange-700/10 text-orange-700 ring-1 ring-inset ring-orange-200 dark:bg-orange-300/15 dark:text-orange-300 dark:ring-orange-700",
    Major:
      "bg-amber-700/10 text-amber-700 ring-1 ring-inset ring-amber-200 dark:bg-amber-300/15 dark:text-amber-300 dark:ring-amber-700",
    Minor:
      "bg-sky-700/10 text-sky-700 ring-1 ring-inset ring-sky-200 dark:bg-sky-300/15 dark:text-sky-300 dark:ring-sky-700",
    Trivial:
      "bg-gray-400/10 text-gray-400 ring-1 ring-inset ring-gray-200 dark:bg-gray-500/15 dark:text-gray-500 dark:ring-gray-600",
  };

  let hasFilters = $derived(
    !!draft ||
      !!filterProject ||
      filterStatuses.join(",") !== ACTIVE_STATUSES.join(",") ||
      filterPriorities.length > 0,
  );
</script>

<Card.Root>
  <Card.CardHeader>
    <Card.CardTitle>Filters</Card.CardTitle>
  </Card.CardHeader>
  <Card.CardContent>
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-1.5">
        <span class="text-muted-foreground text-xs font-medium">Search</span>
        <input
          type="text"
          placeholder="Search title or description..."
          class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
          aria-label="Search"
          bind:value={draft}
        />
      </div>
      {#if !hideProject}
        <div class="flex flex-col gap-1.5">
          <span class="text-muted-foreground text-xs font-medium">Project</span>
          <select
            class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
            value={filterProject}
            onchange={(e) => onProjectChange?.(e.currentTarget.value)}
          >
            <option value="">All projects</option>
            {#each projects as p}
              <option value={p.id}>{p.name}</option>
            {/each}
          </select>
        </div>
      {/if}

      <div class="flex flex-col gap-1.5">
        <span class="text-muted-foreground text-xs font-medium">Status</span>
        <div class="flex flex-wrap gap-1">
          {#each STATUSES as s}
            <button
              type="button"
              onclick={() => onStatusToggle?.(s)}
              class="rounded-md px-2 py-1 text-xs font-medium transition-colors {filterStatuses.includes(
                s,
              )
                ? statusFilterColors[s]
                : 'bg-muted text-muted-foreground hover:bg-muted/80'}"
            >
              {s}
            </button>
          {/each}
        </div>
      </div>

      <div class="flex flex-col gap-1.5">
        <span class="text-muted-foreground text-xs font-medium">Priority</span>
        <div class="flex flex-wrap gap-1">
          {#each PRIORITIES as p}
            <button
              type="button"
              onclick={() => onPriorityToggle?.(p)}
              class="rounded-md px-2 py-1 text-xs font-medium transition-colors {filterPriorities.includes(
                p,
              )
                ? priorityFilterColors[p]
                : 'bg-muted text-muted-foreground hover:bg-muted/80'}"
            >
              {p}
            </button>
          {/each}
        </div>
      </div>

      {#if hasFilters}
        <Button variant="ghost" size="sm" onclick={() => onReset?.()}>
          Clear filters
        </Button>
      {/if}
    </div>
  </Card.CardContent>
</Card.Root>
