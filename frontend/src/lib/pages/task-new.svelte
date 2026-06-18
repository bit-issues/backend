<script lang="ts">
  import { listProjects } from "$lib/api/projects";
  import { createTask } from "$lib/api/tasks";
  import { navigate } from "$lib/router/routes";
  import { touchProject } from "$lib/stores/recent-projects.svelte";
  import TaskForm from "$lib/components/TaskForm.svelte";
  import type { Project } from "$lib/types/api";
  import { onMount } from "svelte";

  let { params = {} }: { params?: Record<string, string> } = $props();

  let projects = $state<Project[]>([]);
  let loading = $state(true);

  let initialProject =
    new URLSearchParams(
      window.location.hash.split("?")[1] || window.location.search,
    ).get("project") || "";

  let initialProjectSlug = $state("");

  onMount(() => {
    loading = true;
    listProjects(100, 0)
      .then((r) => {
        projects = r.items;
        if (initialProject) {
          const match = projects.find((p) => p.id === initialProject);
          if (match) initialProjectSlug = match.id;
        }
      })
      .catch(() => {})
      .finally(() => {
        loading = false;
      });
  });

  async function handleSubmit(data: {
    project_slug: string;
    title: string;
    description: string;
    priority: string;
    kind: string;
    assignee_id: number | null | undefined;
    due_date: string;
  }) {
    const task = await createTask({
      project_slug: data.project_slug,
      title: data.title,
      description: data.description || undefined,
      priority: data.priority as any,
      kind: data.kind as any,
      assignee_id: data.assignee_id ?? undefined,
      due_date: data.due_date || undefined,
    });
    const match = projects.find((p) => p.id === data.project_slug);
    touchProject({
      id: data.project_slug,
      name: match?.name || data.project_slug,
    });
    navigate(`/tasks/${task.id}`);
  }
</script>

<div class="mx-auto max-w-3xl">
  {#if loading}
    <p class="text-muted-foreground py-4 text-center text-sm">Loading...</p>
  {:else}
    <TaskForm
      mode="create"
      {projects}
      initialProjectSlug={initialProjectSlug}
      onSubmit={handleSubmit}
    />
  {/if}
</div>
