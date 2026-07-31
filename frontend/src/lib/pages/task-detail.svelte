<script lang="ts">
  import { getTaskByProjectAndNumber, deleteTask, updateTask } from "$lib/api/tasks";
  import {
    createComment,
    updateComment,
    deleteComment,
  } from "$lib/api/comments";
  import { deleteAttachment } from "$lib/api/attachments";
  import { navigate } from "$lib/router/routes";
  import { touchProject } from "$lib/stores/recent-projects.svelte";
  import { getUser } from "$lib/stores/auth.svelte";
  import AssigneeCombobox from "$lib/components/AssigneeCombobox.svelte";
  import CommentList from "$lib/components/CommentList.svelte";
  import CommentForm from "$lib/components/CommentForm.svelte";
  import AttachmentList from "$lib/components/AttachmentList.svelte";
  import AttachmentUpload from "$lib/components/AttachmentUpload.svelte";
  import { Textarea } from "$lib/components/ui/textarea";
  import { Button } from "$lib/components/ui/button";
  import { Separator } from "$lib/components/ui/separator";
  import * as Card from "$lib/components/ui/card";
  import { parse } from "marked";
  import DOMPurify from "dompurify";
  import type { TaskDetails, Attachment } from "$lib/types/api";
  import { STATUSES } from "$lib/types/api";
  import { processAutoLinks } from "$lib/autolink";
  import { resolveAttachmentRefs } from "$lib/resolve-attachment-refs";
  import type { AutoLinkContext } from "$lib/autolink";
  import { toast } from "$lib/toast";

  let { params = {} }: { params?: Record<string, string> } = $props();
  let slug = $derived(params.slug || "");
  let number = $derived(Number(params.number));

  let task = $state<TaskDetails | null>(null);
  let taskId = $state(0);
  let loading = $state(true);
  let error = $state("");

  let deleting = $state(false);

  let selectedStatus = $state("New");
  let statusComment = $state("");
  let savingStatus = $state(false);

  let assigneeId = $state<number | null>(null);
  let assigning = false;
  let ready = $state(false);

  let currentUserId = $derived(getUser()?.id);

  let autoLinkCtx = $derived<AutoLinkContext>({
    repoUrl: task?.project.repo_url || undefined,
  });

  let attachmentUrls = $derived(
    new Map(task?.attachments?.map((a) => [a.id, a.download_url]) ?? []),
  );

  let renderedDescription = $derived(
    task?.description
      ? DOMPurify.sanitize(
          parse(
            processAutoLinks(
              resolveAttachmentRefs(task.description, attachmentUrls),
              autoLinkCtx,
            ),
            { async: false },
          ) as string,
        )
      : "",
  );

  $effect(() => {
    const currentSlug = slug;
    const currentNumber = number;
    if (!currentSlug || !currentNumber) return;

    loading = true;
    error = "";
    task = null;
    ready = false;
    taskId = 0;

    getTaskByProjectAndNumber(currentSlug, currentNumber)
      .then((t) => {
        if (currentSlug !== slug || currentNumber !== number) return;
        task = t;
        taskId = t.id;
        selectedStatus = t.status;
        assigneeId = t.assignee?.id ?? null;
        ready = true;
        touchProject({ id: t.project.id, name: t.project.name });
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

  $effect(() => {
    if (!ready || assigning || !task) return;
    const newId = assigneeId;
    const oldId = task.assignee?.id ?? null;
    if (newId !== oldId) {
      assigning = true;
      updateTask(taskId, { assignee_id: newId ?? 0 })
        .then((updated) => {
          task = updated;
          assigneeId = updated.assignee?.id ?? null;
        })
        .catch((e: any) => {
          toast.error(e.message || "Failed to update assignee");
          assigneeId = oldId;
        })
        .finally(() => {
          assigning = false;
        });
    }
  });

  async function handleDelete() {
    if (!confirm("Delete this task permanently?")) return;
    deleting = true;
    try {
      await deleteTask(taskId);
      navigate("/dashboard");
    } catch (e: any) {
      toast.error(e.message || "Failed to delete task");
    } finally {
      deleting = false;
    }
  }

  async function handleAddComment(content: string) {
    await createComment(taskId, { content });
    // Refetch the task so the freshly-confirmed inline attachment is present in
    // task.attachments (attachmentUrls derives from it), letting the newly
    // posted comment's inline image resolve without a manual page refresh.
    const refreshed = await getTaskByProjectAndNumber(slug, number);
    task = refreshed;
  }

  async function handleEditComment(commentId: number, content: string) {
    await updateComment(taskId, commentId, { content });
    const refreshed = await getTaskByProjectAndNumber(slug, number);
    task = refreshed;
  }

  async function handleDeleteComment(commentId: number) {
    await deleteComment(taskId, commentId);
    task = {
      ...task!,
      comments: task!.comments.filter((c) => c.id !== commentId),
    };
  }

  async function handleDeleteAttachment(attachmentId: number) {
    await deleteAttachment(taskId, attachmentId);
    task = {
      ...task!,
      attachments: task!.attachments.filter((a) => a.id !== attachmentId),
    };
  }

  async function handleStatusChange() {
    savingStatus = true;
    try {
      const updated = await updateTask(taskId, {
        status: selectedStatus,
        comment: statusComment || undefined,
      });
      task = { ...updated };
      statusComment = "";
    } catch (e: any) {
      toast.error(e.message || "Failed to update status");
    } finally {
      savingStatus = false;
    }
  }

  function formatDate(d: string | null): string {
    if (!d) return "—";
    return new Date(d).toLocaleDateString();
  }

  function formatDateTime(d: string): string {
    return new Date(d).toLocaleString();
  }
</script>

<div class="mx-auto max-w-7xl space-y-6">
  {#if loading}
    <p class="text-muted-foreground py-8 text-center">Loading...</p>
  {:else if error}
    <p class="text-destructive py-8 text-center">{error}</p>
  {:else if task}
    <div class="flex items-start justify-between gap-4">
      <div>
        <p class="text-muted-foreground mb-1 text-sm">
          <a
            href="#/projects"
            class="hover:text-foreground underline underline-offset-2"
            >Projects</a
          >
          /
          <a
            href={`#/projects/${task.project.id}`}
            class="hover:text-foreground underline underline-offset-2"
            >{task.project.name}</a
          >
          /
          <span class="text-muted-foreground font-mono">#{task.number}</span>
        </p>
        <h1 class="text-2xl font-semibold">{task.title}</h1>
      </div>
      <div class="flex shrink-0 gap-2">
        <Button variant="outline" onclick={() => navigate(`/tasks/${slug}/${number}/edit`)}
          >Edit</Button
        >
        <Button
          variant="destructive"
          onclick={handleDelete}
          disabled={deleting}
        >
          {deleting ? "Deleting..." : "Delete"}
        </Button>
      </div>
    </div>

    <div class="grid grid-cols-[1fr_280px] gap-6">
      <div class="min-w-0 space-y-6">
        <div>
          <h2 class="mb-2 text-sm font-medium text-muted-foreground">
            Description
          </h2>
          {#if task.description}
            <div class="prose prose-sm dark:prose-invert max-w-none">
              {@html renderedDescription}
            </div>
          {:else}
            <p class="text-muted-foreground italic">No description</p>
          {/if}
        </div>

        <Separator />

        <div>
          <h2 class="mb-3 text-sm font-medium text-muted-foreground">
            Comments ({task.comments.length})
          </h2>

          <div>
            <CommentList
              comments={task.comments}
              {currentUserId}
              taskId={task.id}
              attachmentsMap={attachmentUrls}
              {autoLinkCtx}
              onEdit={handleEditComment}
              onDelete={handleDeleteComment}
            />
            <div class="mt-4">
              <CommentForm taskId={task.id} onSubmit={handleAddComment} />
            </div>
          </div>
        </div>
      </div>

      <aside class="space-y-4">
        <Card.Root>
          <Card.CardHeader>
            <Card.CardTitle>Details</Card.CardTitle>
          </Card.CardHeader>
          <Card.CardContent>
            <div class="flex flex-col gap-3">
              <div class="flex flex-col gap-1.5">
                <span class="text-muted-foreground text-xs font-medium"
                  >Status</span
                >
                <select
                  bind:value={selectedStatus}
                  class="w-fit rounded-md border bg-background px-2 py-1 text-xs"
                >
                  {#each STATUSES as s}
                    <option value={s}>{s}</option>
                  {/each}
                </select>
                {#if selectedStatus !== task.status}
                  <Textarea
                    bind:value={statusComment}
                    placeholder="Reason for change (optional)"
                    class="min-h-16 text-xs"
                  />
                  <Button
                    size="sm"
                    onclick={handleStatusChange}
                    disabled={savingStatus}
                  >
                    {savingStatus ? "Saving..." : "Change"}
                  </Button>
                {/if}
              </div>
              <div class="flex flex-col gap-1.5">
                <span class="text-muted-foreground text-xs font-medium"
                  >Priority</span
                >
                <span
                  class="w-fit inline-flex items-center rounded-md border bg-clip-padding px-2 py-0.5 text-xs font-semibold border-amber-300 text-amber-700 dark:border-amber-600 dark:text-amber-400"
                >
                  {task.priority}
                </span>
              </div>
              <div class="flex flex-col gap-1.5">
                <span class="text-muted-foreground text-xs font-medium"
                  >Kind</span
                >
                <span
                  class="w-fit inline-flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400"
                >
                  <span
                    class="inline-block size-1.5 rounded-full bg-emerald-500"
                  ></span>
                  {task.kind}
                </span>
              </div>
              <Separator />
              <div class="flex flex-col gap-1.5">
                <span class="text-muted-foreground text-xs font-medium"
                  >Assignee</span
                >
                <AssigneeCombobox
                  bind:value={assigneeId}
                  initialName={task.assignee?.name || ""}
                  placeholder="Search assignee..."
                />
              </div>
              <div class="flex flex-col gap-1.5">
                <span class="text-muted-foreground text-xs font-medium"
                  >Due date</span
                >
                <span class="text-sm">{formatDate(task.due_date)}</span>
              </div>
              <Separator />
              <div class="flex flex-col gap-1.5">
                <span class="text-muted-foreground text-xs font-medium"
                  >Created</span
                >
                <span class="text-sm">{formatDateTime(task.created_at)}</span>
              </div>
              <div class="flex flex-col gap-1.5">
                <span class="text-muted-foreground text-xs font-medium"
                  >Author</span
                >
                <span class="text-sm">{task.author?.name || "—"}</span>
              </div>
            </div>
          </Card.CardContent>
        </Card.Root>

        <Card.Root>
          <Card.CardHeader>
            <Card.CardTitle
              >Attachments ({task.attachments.length})</Card.CardTitle
            >
          </Card.CardHeader>
          <Card.CardContent>
            <AttachmentList
              attachments={task.attachments}
              {currentUserId}
              taskAuthorId={task.author?.id ?? 0}
              onDelete={handleDeleteAttachment}
            />
            <AttachmentUpload
              taskId={taskId}
              onUploaded={(confirmed) => {
                const att: Attachment = {
                  id: confirmed.id,
                  task_id: taskId,
                  uploaded_by: confirmed.uploaded_by,
                  file_name: confirmed.file_name,
                  size_bytes: confirmed.size_bytes,
                  download_url: confirmed.download_url,
                  created_at: confirmed.uploaded_at,
                  updated_at: confirmed.uploaded_at,
                };
                task = {
                  ...task!,
                  attachments: [...task!.attachments, att],
                };
              }}
            />
          </Card.CardContent>
        </Card.Root>
      </aside>
    </div>
  {/if}
</div>
