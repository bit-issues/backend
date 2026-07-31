<script lang="ts">
  import { getTaskByProjectAndNumber, updateTask } from "$lib/api/tasks";
  import { listProjects } from "$lib/api/projects";
  import { navigate } from "$lib/router/routes";
  import TaskForm from "$lib/components/TaskForm.svelte";
  import type { TaskDetails, Project } from "$lib/types/api";

  let { params = {} }: { params?: Record<string, string> } = $props();
  let slug = $derived(params.slug || "");
  let number = $derived(Number(params.number));

  let task = $state<TaskDetails | null>(null);
  let taskId = $state(0);
  let projects = $state<Project[]>([]);
  let loading = $state(true);
  let error = $state("");

  $effect(() => {
    const currentSlug = slug;
    const currentNumber = number;
    if (!currentSlug || !currentNumber) return;

    loading = true;
    error = "";
    task = null;
    taskId = 0;

    Promise.all([
      getTaskByProjectAndNumber(currentSlug, currentNumber),
      listProjects(100, 0),
    ])
      .then(([t, p]) => {
        if (currentSlug !== slug || currentNumber !== number) return;
        task = t;
        taskId = t.id;
        projects = p.items;
      })
      .catch((e) => {
        if (currentSlug !== slug || currentNumber !== number) return;
        error = e.message || "Failed to load task";
      })
      .finally(() => {
        if (currentSlug !== slug || currentNumber !== number) return;
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
    const updated = await updateTask(taskId, {
      title: data.title,
      description: data.description || undefined,
      priority: data.priority as any,
      kind: data.kind as any,
      status: data.status as any,
      assignee_id: data.assignee_id ?? undefined,
      due_date: data.due_date,
    });
    navigate(`/tasks/${updated.project_slug}/${updated.number}`);
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
      {taskId}
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
