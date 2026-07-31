<script lang="ts">
  import { untrack } from "svelte";
  import { Textarea } from "$lib/components/ui/textarea";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import AssigneeCombobox from "$lib/components/AssigneeCombobox.svelte";
  import * as Card from "$lib/components/ui/card";
  import { PRIORITIES, KINDS, STATUSES, type Project } from "$lib/types/api";
  import InlineImageButton from "$lib/components/InlineImageButton.svelte";
  import { InlineImageEditor } from "$lib/inline-image-editor.svelte";

  interface TaskFormData {
    project_slug: string;
    title: string;
    description: string;
    priority: string;
    kind: string;
    status?: string;
    assignee_id: number | null | undefined;
    due_date: string;
  }

  interface Props {
    mode?: "create" | "edit";
    taskId?: number;
    projects?: Project[];
    initialTitle?: string;
    initialDescription?: string;
    initialPriority?: string;
    initialKind?: string;
    initialStatus?: string;
    initialAssigneeId?: number | null;
    initialDueDate?: string;
    initialProjectSlug?: string;
    initialAssigneeName?: string;
    taskNumber?: number;
    projectName?: string;
    onSubmit: (data: TaskFormData) => Promise<void>;
  }

  let {
    mode = "create",
    taskId = 0,
    projects = [] as Project[],
    initialTitle = "",
    initialDescription = "",
    initialPriority = "Major",
    initialKind = "Bug",
    initialStatus = "Open",
    initialAssigneeId = null as number | null,
    initialDueDate = "",
    initialProjectSlug = "",
    initialAssigneeName = "",
    taskNumber,
    projectName,
    onSubmit,
  }: Props = $props();

  let title = $state("");
  let description = $state("");
  let priority = $state("Major");
  let kind = $state("Bug");
  let status = $state("Open");
  let assigneeId = $state<number | null>(null);
  let dueDate = $state("");
  let projectSlug = $state("");

  $effect(() => {
    title = untrack(() => initialTitle);
    description = untrack(() => initialDescription);
    priority = untrack(() => initialPriority);
    kind = untrack(() => initialKind);
    status = untrack(() => initialStatus);
    assigneeId = untrack(() => initialAssigneeId);
    dueDate = untrack(() => initialDueDate);
    projectSlug = untrack(() => initialProjectSlug);
  });

  let saving = $state(false);
  let error = $state("");
  let descriptionTextarea: HTMLTextAreaElement | undefined = $state();
  let fileInput: HTMLInputElement | undefined = $state();

  const images = new InlineImageEditor({
    getTaskId: () => taskId,
    getValue: () => description,
    setValue: (v) => {
      description = v;
    },
    getTextarea: () => descriptionTextarea,
  });

  async function handleSubmit() {
    if (!title.trim()) {
      error = "Title is required";
      return;
    }
    if (!projectSlug) {
      error = "Project is required";
      return;
    }
    if (images.uploading) return;
    saving = true;
    error = "";

    try {
      await onSubmit({
        project_slug: projectSlug,
        title: title.trim(),
        description: description.trim() || "",
        priority,
        kind,
        status: mode === "edit" ? status : undefined,
        assignee_id: assigneeId ?? undefined,
        due_date: dueDate || "",
      });
    } catch (e: any) {
      error = e.message || "Failed to save task";
    } finally {
      saving = false;
    }
  }
</script>

<Card.Root class="overflow-visible">
  <Card.CardHeader>
    <Card.CardTitle>{mode === "edit" ? "Edit Task" : "New Task"}</Card.CardTitle
    >
    <Card.CardDescription>
      {#if mode === "edit" && taskNumber != null}
        <span class="font-mono">#{taskNumber}</span>
        {#if projectName}
          &middot; {projectName}
        {/if}
      {:else}
        Create a new task in a project
      {/if}
    </Card.CardDescription>
  </Card.CardHeader>
  <Card.CardContent>
    <form
      class="space-y-4"
      onsubmit={(e) => {
        e.preventDefault();
        handleSubmit();
      }}
    >
      <div class="space-y-1.5">
        <label for="project" class="text-sm font-medium"
          >Project <span class="text-destructive">*</span></label
        >
        <select
          id="project"
          class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
          bind:value={projectSlug}
          disabled={mode === "edit"}
        >
          <option value="">Select a project</option>
          {#each projects as p}
            <option value={p.id}>{p.name}</option>
          {/each}
        </select>
      </div>

      <div class="space-y-1.5">
        <label for="title" class="text-sm font-medium"
          >Title <span class="text-destructive">*</span></label
        >
        <Input id="title" bind:value={title} placeholder="Task title" />
      </div>

      <div class="space-y-1.5">
        <div class="flex items-center justify-between">
          <label for="description" class="text-sm font-medium"
            >Description</label
          >
          {#if taskId > 0}
            <InlineImageButton
              uploading={images.uploading}
              onclick={() => fileInput?.click()}
            />
          {:else}
            <span class="text-xs text-muted-foreground"
              >Image upload available when editing</span
            >
          {/if}
        </div>
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          ondragover={(e) => images.handleDragOver(e)}
          ondrop={(e) => images.handleImageDrop(e)}
        >
          <Textarea
            id="description"
            bind:value={description}
            bind:this={descriptionTextarea}
            onpaste={(e) => images.handleImagePaste(e)}
            placeholder="Optional description (supports Markdown)"
            class="min-h-32"
          />
        </div>
        <input
          type="file"
          bind:this={fileInput}
          accept="image/png,image/jpeg,image/gif,image/webp"
          class="hidden"
          onchange={(e) => images.handleImagePick(e)}
        />
      </div>

      <div class="grid grid-cols-2 gap-4" class:grid-cols-3={mode === "edit"}>
        {#if mode === "edit"}
          <div class="space-y-1.5">
            <label for="status" class="text-sm font-medium">Status</label>
            <select
              id="status"
              class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
              bind:value={status}
            >
              {#each STATUSES as s}
                <option value={s}>{s}</option>
              {/each}
            </select>
          </div>
        {/if}

        <div class="space-y-1.5">
          <label for="priority" class="text-sm font-medium">Priority</label>
          <select
            id="priority"
            class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
            bind:value={priority}
          >
            {#each PRIORITIES as p}
              <option value={p}>{p}</option>
            {/each}
          </select>
        </div>

        <div class="space-y-1.5">
          <label for="kind" class="text-sm font-medium">Kind</label>
          <select
            id="kind"
            class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
            bind:value={kind}
          >
            {#each KINDS as k}
              <option value={k}>{k}</option>
            {/each}
          </select>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div class="space-y-1.5">
          <label for="assignee" class="text-sm font-medium">Assignee</label>
          <AssigneeCombobox
            id="assignee"
            bind:value={assigneeId}
            initialName={initialAssigneeName}
            placeholder="Search user..."
          />
        </div>

        <div class="space-y-1.5">
          <label for="dueDate" class="text-sm font-medium">Due date</label>
          <div class="relative">
            <Input
              id="dueDate"
              type="date"
              bind:value={dueDate}
              class="w-full pr-8"
            />
            {#if dueDate}
              <button
                type="button"
                onclick={() => (dueDate = "")}
                class="absolute right-1 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground inline-flex cursor-pointer items-center justify-center text-lg leading-none"
              >
                &times;
              </button>
            {/if}
          </div>
        </div>
      </div>

      {#if error}
        <p class="text-destructive text-sm">{error}</p>
      {/if}

      <div class="flex gap-2 pt-2">
        <Button type="submit" disabled={saving || images.uploading}>
          {saving
            ? mode === "edit"
              ? "Saving..."
              : "Creating..."
            : mode === "edit"
              ? "Save"
              : "Create"}
        </Button>
        <Button type="button" variant="outline" onclick={() => history.back()}
          >Cancel</Button
        >
      </div>
    </form>
  </Card.CardContent>
</Card.Root>
