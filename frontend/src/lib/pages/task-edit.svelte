<script lang="ts">
  import { getTask, updateTask } from "$lib/api/tasks";
  import { listProjects } from "$lib/api/projects";
  import { navigate } from "$lib/router/routes";
  import TaskForm from "$lib/components/TaskForm.svelte";
  import type { TaskDetails, Project } from "$lib/types/api";
  import { onMount } from "svelte";

  let { params = {} }: { params?: Record<string, string> } = $props();
  let id = $derived(Number(params.id));

  let task = $state<TaskDetails | null>(null);
  let projects = $state<Project[]>([]);
  let loading = $state(true);
  let error = $state("");

  onMount(() => {
    if (!id) return;
    Promise.all([getTask(id), listProjects(100, 0)])
      .then(([t, p]) => {
        task = t;
        projects = p.items;
      })
      .catch((e) => {
        error = e.message || "Failed to load task";
      })
      .finally(() => {
        loading = false;
      });
  });

  async function handleSubmit(data: {
    title: string;
    description: string;
    priority: string;
    kind: string;
    status?: string;
    assignee_id: number | null | undefined;
    due_date: string;
  }) {
    const updated = await updateTask(id, {
      title: data.title,
      description: data.description || undefined,
      priority: data.priority as any,
      kind: data.kind as any,
      status: data.status as any,
      assignee_id: data.assignee_id ?? undefined,
      due_date: data.due_date,
    });
    navigate(`/tasks/${updated.id}`);
  }
</script>

<div class="mx-auto max-w-3xl">
  {#if loading}
    <p class="text-muted-foreground py-4 text-center text-sm">Loading...</p>
  {:else if error && !task}
    <p class="text-destructive py-4 text-center text-sm">{error}</p>
  {:else if task}
    <TaskForm
      mode="edit"
      {projects}
      initialTitle={task.title}
      initialDescription={task.description || ""}
      initialPriority={task.priority}
      initialKind={task.kind}
      initialStatus={task.status}
      initialAssigneeId={task.assignee?.id ?? null}
      initialAssigneeName={task.assignee?.name ?? ""}
      initialDueDate={task.due_date || ""}
      initialProjectSlug={task.project.id}
      taskNumber={task.number}
      projectName={task.project.name}
      onSubmit={handleSubmit}
    />
  {/if}
</div>
