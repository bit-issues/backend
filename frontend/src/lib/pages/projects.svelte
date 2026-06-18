<script lang="ts">
  import { listProjects } from "$lib/api/projects";
  import { navigate } from "$lib/router/routes";
  import {
    enrichRecentFromProjects,
    touchProject,
  } from "$lib/stores/recent-projects.svelte";
  import * as Card from "$lib/components/ui/card";
  import { Button } from "$lib/components/ui/button";
  import { Separator } from "$lib/components/ui/separator";
  import FolderKanbanIcon from "@lucide/svelte/icons/folder-kanban";
  import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
  import type { Project } from "$lib/types/api";

  let { params = {} }: { params?: Record<string, string> } = $props();

  let projects = $state<Project[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state("");
  let offset = $state(0);
  const limit = 20;

  function load() {
    loading = true;
    error = "";
    listProjects(limit, offset)
      .then((res) => {
        projects = res.items;
        total = res.total;
        enrichRecentFromProjects(res.items);
      })
      .catch((e) => {
        error = e.message || "Failed to load projects";
      })
      .finally(() => {
        loading = false;
      });
  }

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

  {#if loading}
    <p class="text-muted-foreground py-8 text-center">Loading...</p>
  {:else if error}
    <p class="text-destructive py-8 text-center">{error}</p>
  {:else if projects.length === 0}
    <Card.Root class="py-12">
      <Card.CardContent class="flex flex-col items-center gap-2">
        <FolderKanbanIcon class="text-muted-foreground size-12" />
        <p class="text-muted-foreground text-sm">No projects yet</p>
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
