<script lang="ts">
  import { getTask, deleteTask, updateTask } from "$lib/api/tasks";
  import {
    createComment,
    updateComment,
    deleteComment,
  } from "$lib/api/comments";
  import { deleteAttachment } from "$lib/api/attachments";
  import { initUpload, confirmUpload } from "$lib/api/attachments";
  import { navigate } from "$lib/router/routes";
  import { getUser } from "$lib/stores/auth.svelte";
  import AssigneeCombobox from "$lib/components/AssigneeCombobox.svelte";
  import { Textarea } from "$lib/components/ui/textarea";
  import { Button } from "$lib/components/ui/button";
  import { Separator } from "$lib/components/ui/separator";
  import * as Card from "$lib/components/ui/card";
  import { parse } from "marked";
  import { onMount } from "svelte";
  import type { TaskDetails, Comment } from "$lib/types/api";
  import { STATUSES } from "$lib/types/api";
  import { processAutoLinks } from "$lib/autolink";
  import type { AutoLinkContext } from "$lib/autolink";

  let { params = {} }: { params?: Record<string, string> } = $props();
  let id = $derived(Number(params.id));

  let task = $state<TaskDetails | null>(null);
  let loading = $state(true);
  let error = $state("");

  let newComment = $state("");
  let posting = $state(false);

  let editingCommentId = $state<number | null>(null);
  let editCommentText = $state("");
  let savingComment = $state(false);

  let deleting = $state(false);

  let selectedStatus = $state("New");
  let statusComment = $state("");
  let savingStatus = $state(false);

  let assigneeId = $state<number | null>(null);
  let assigning = $state(false);
  let ready = $state(false);

  let currentUserId = $derived(getUser()?.id);

  let autoLinkCtx = $derived<AutoLinkContext>({
    repoUrl: task?.project.repo_url || undefined,
  });

  let renderedDescription = $derived(
    task?.description
      ? (parse(processAutoLinks(task.description, autoLinkCtx), {
          async: false,
        }) as string)
      : "",
  );

  onMount(() => {
    if (!id) return;
    getTask(id)
      .then((t) => {
        task = t;
        selectedStatus = t.status;
        assigneeId = t.assignee?.id ?? null;
        ready = true;
      })
      .catch((e) => {
        error = e.message || "Failed to load task";
      })
      .finally(() => {
        loading = false;
      });
  });

  $effect(() => {
    if (!ready || assigning || !task) return;
    const newId = assigneeId;
    const oldId = task.assignee?.id ?? null;
    if (newId !== oldId) {
      assigning = true;
      updateTask(id, { assignee_id: newId ?? 0 })
        .then((updated) => {
          task = updated;
          assigneeId = updated.assignee?.id ?? null;
        })
        .catch((e: any) => {
          alert(e.message || "Failed to update assignee");
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
      await deleteTask(id);
      navigate("/dashboard");
    } catch (e: any) {
      alert(e.message || "Failed to delete task");
    } finally {
      deleting = false;
    }
  }

  async function handleAddComment() {
    if (!newComment.trim()) return;
    posting = true;
    try {
      const created = await createComment(id, { content: newComment });
      task = {
        ...task!,
        comments: [...task!.comments, created as Comment],
      };
      newComment = "";
    } catch (e: any) {
      alert(e.message || "Failed to add comment");
    } finally {
      posting = false;
    }
  }

  function startEdit(comment: Comment) {
    editingCommentId = comment.id;
    editCommentText = comment.content;
  }

  function cancelEdit() {
    editingCommentId = null;
    editCommentText = "";
  }

  async function handleEditComment(commentId: number) {
    if (!editCommentText.trim()) return;
    savingComment = true;
    try {
      const updated = await updateComment(id, commentId, {
        content: editCommentText,
      });
      task = {
        ...task!,
        comments: task!.comments.map((c) =>
          c.id === commentId ? (updated as Comment) : c,
        ),
      };
      editingCommentId = null;
      editCommentText = "";
    } catch (e: any) {
      alert(e.message || "Failed to update comment");
    } finally {
      savingComment = false;
    }
  }

  async function handleDeleteComment(commentId: number) {
    if (!confirm("Delete this comment?")) return;
    try {
      await deleteComment(id, commentId);
      task = {
        ...task!,
        comments: task!.comments.filter((c) => c.id !== commentId),
      };
    } catch (e: any) {
      alert(e.message || "Failed to delete comment");
    }
  }

  async function handleDeleteAttachment(attachmentId: number) {
    if (!confirm("Delete this attachment?")) return;
    try {
      await deleteAttachment(id, attachmentId);
      task = {
        ...task!,
        attachments: task!.attachments.filter((a) => a.id !== attachmentId),
      };
    } catch (e: any) {
      alert(e.message || "Failed to delete attachment");
    }
  }

  async function handleUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    try {
      const init = await initUpload(id, {
        file_name: file.name,
        size_bytes: file.size,
      });
      try {
        const res = await fetch(init.upload_url, { method: "PUT", body: file });
        if (!res.ok) throw new Error(`Upload failed with status ${res.status}`);
      } catch {
        alert("Upload failed: S3 storage not available in dev mode");
        input.value = "";
        return;
      }
      const confirmed = await confirmUpload(id, init.id);
      task = {
        ...task!,
        attachments: [
          ...task!.attachments,
          {
            id: confirmed.id,
            task_id: id,
            author: getUser()!,
            file_name: confirmed.file_name,
            size_bytes: confirmed.size_bytes,
            status: "uploaded",
            created_at: "",
            updated_at: "",
          },
        ],
      };
    } catch (e: any) {
      alert(e.message || "Upload failed");
    }
    input.value = "";
  }

  async function handleStatusChange() {
    savingStatus = true;
    try {
      const updated = await updateTask(id, {
        status: selectedStatus,
        comment: statusComment || undefined,
      });
      task = { ...updated };
      statusComment = "";
    } catch (e: any) {
      alert(e.message || "Failed to update status");
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

  function renderMarkdown(text: string): string {
    if (!text) return "";
    return parse(processAutoLinks(text, autoLinkCtx), {
      async: false,
    }) as string;
  }

  function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
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
          <button
            class="hover:text-foreground cursor-pointer underline underline-offset-2"
            onclick={() => navigate("/projects")}>Projects</button
          >
          /
          <button
            class="hover:text-foreground cursor-pointer underline underline-offset-2"
            onclick={() => navigate(`/projects/${task?.project.id}`)}
            >{task.project.name}</button
          >
          /
          <span class="text-muted-foreground font-mono">#{task.number}</span>
        </p>
        <h1 class="text-2xl font-semibold">{task.title}</h1>
      </div>
      <div class="flex shrink-0 gap-2">
        <Button variant="outline" onclick={() => navigate(`/tasks/${id}/edit`)}
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

          <div class="space-y-3">
            {#each task.comments as comment (comment.id)}
              <div class="rounded-lg border p-3">
                <div
                  class="mb-1 flex items-center gap-2 text-xs text-muted-foreground"
                >
                  <span class="font-medium text-foreground"
                    >{comment.author.name}</span
                  >
                  &middot;
                  <span>{formatDateTime(comment.created_at)}</span>
                  {#if comment.updated_at !== comment.created_at}
                    &middot;
                    <span class="italic">edited</span>
                  {/if}
                </div>

                {#if editingCommentId === comment.id}
                  <Textarea
                    bind:value={editCommentText}
                    class="mb-2 min-h-16"
                  />
                  <div class="flex gap-2">
                    <Button
                      size="sm"
                      onclick={() => handleEditComment(comment.id)}
                      disabled={savingComment}
                    >
                      {savingComment ? "Saving..." : "Save"}
                    </Button>
                    <Button size="sm" variant="ghost" onclick={cancelEdit}
                      >Cancel</Button
                    >
                  </div>
                {:else}
                  <div
                    class="prose prose-sm dark:prose-invert max-w-none text-sm"
                  >
                    {@html renderMarkdown(comment.content)}
                  </div>
                  {#if comment.author.id === currentUserId}
                    <div class="mt-1 flex gap-2">
                      <button
                        class="text-muted-foreground hover:text-foreground cursor-pointer text-xs underline underline-offset-2"
                        onclick={() => startEdit(comment)}>Edit</button
                      >
                      <button
                        class="text-muted-foreground hover:text-destructive cursor-pointer text-xs underline underline-offset-2"
                        onclick={() => handleDeleteComment(comment.id)}
                        >Delete</button
                      >
                    </div>
                  {/if}
                {/if}
              </div>
            {:else}
              <p class="text-muted-foreground text-sm italic">
                No comments yet
              </p>
            {/each}
          </div>

          <div class="mt-4">
            <Textarea
              bind:value={newComment}
              placeholder="Add a comment..."
              class="mb-2 min-h-20"
            />
            <Button
              onclick={handleAddComment}
              disabled={posting || !newComment.trim()}
            >
              {posting ? "Posting..." : "Comment"}
            </Button>
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
            {#if task.attachments.length > 0}
              <div class="mb-3 space-y-1">
                {#each task.attachments as att (att.id)}
                  <div
                    class="flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm"
                  >
                    <span class="text-muted-foreground text-lg">&#128206;</span>
                    {#if att.download_url}
                      <a
                        href={att.download_url}
                        target="_blank"
                        rel="noreferrer"
                        class="flex-1 truncate hover:underline cursor-pointer"
                        >{att.file_name}</a
                      >
                    {:else}
                      <span class="flex-1 truncate">{att.file_name}</span>
                    {/if}
                    <span class="text-muted-foreground shrink-0 text-xs"
                      >{formatBytes(att.size_bytes)}</span
                    >
                    <button
                      type="button"
                      onclick={() => handleDeleteAttachment(att.id)}
                      class="text-muted-foreground hover:text-destructive shrink-0 cursor-pointer text-xs underline underline-offset-2"
                    >
                      Delete
                    </button>
                  </div>
                {/each}
              </div>
            {:else}
              <p class="text-muted-foreground mb-3 text-sm italic">
                No attachments
              </p>
            {/if}

            <label
              class="inline-flex cursor-pointer items-center justify-center rounded-lg border border-border bg-background px-2.5 py-1.5 text-sm font-medium text-foreground hover:bg-muted dark:bg-input/30 dark:border-input dark:hover:bg-input/50"
            >
              Upload file
              <input type="file" class="hidden" onchange={handleUpload} />
            </label>
          </Card.CardContent>
        </Card.Root>
      </aside>
    </div>
  {/if}
</div>
