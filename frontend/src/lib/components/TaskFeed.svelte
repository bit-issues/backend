<script lang="ts">
  import { getTaskFeed } from "$lib/api/tasks";
  import {
    createComment,
    updateComment,
    deleteComment,
  } from "$lib/api/comments";
  import { Avatar, AvatarFallback } from "$lib/components/ui/avatar";
  import { Button } from "$lib/components/ui/button";
  import CommentForm from "$lib/components/CommentForm.svelte";
  import CommentItem from "$lib/components/CommentItem.svelte";
  import type { AutoLinkContext } from "$lib/autolink";
  import type { FeedItem, HistoryChange } from "$lib/types/api";
  import { toast } from "$lib/toast";

  const PAGE_SIZE = 20;

  const FIELD_LABELS: Record<string, string> = {
    title: "Title",
    description: "Description",
    priority: "Priority",
    status: "Status",
    kind: "Kind",
    assignee: "Assignee",
    due_date: "Due date",
  };

  let {
    slug = "",
    number = 0,
    taskId = 0,
    currentUserId = 0,
    attachmentsMap = new Map<number, string>(),
    autoLinkCtx = {} as AutoLinkContext,
    refreshKey = 0,
    onCommentAdded = async () => {},
  }: {
    slug: string;
    number: number;
    taskId: number;
    currentUserId: number;
    attachmentsMap: Map<number, string>;
    autoLinkCtx: AutoLinkContext;
    refreshKey?: number;
    onCommentAdded?: () => Promise<void>;
  } = $props();

  let items = $state<FeedItem[]>([]);
  let total = $state(0);
  let nextOffset = $state(0);
  let loading = $state(true);
  let loadingMore = $state(false);
  let error = $state("");
  let generation = 0;

  $effect(() => {
    void refreshKey;

    const currentSlug = slug;
    const currentNumber = number;
    if (!currentSlug || !currentNumber) {
      loading = false;
      return;
    }

    void loadFirstPage(currentSlug, currentNumber);
  });

  async function loadFirstPage(currentSlug: string, currentNumber: number) {
    const requestGeneration = ++generation;
    loading = true;
    loadingMore = false;
    error = "";

    try {
      const response = await getTaskFeed(
        currentSlug,
        currentNumber,
        PAGE_SIZE,
        0,
      );
      if (requestGeneration !== generation) return;

      items = response.items;
      total = response.total;
      nextOffset = response.items.length;
    } catch (e) {
      if (requestGeneration !== generation) return;

      error = e instanceof Error ? e.message : "Failed to load activity";
    } finally {
      if (requestGeneration === generation) {
        loading = false;
      }
    }
  }

  async function loadMore() {
    if (loading || loadingMore || nextOffset >= total) return;

    const requestGeneration = generation;
    loadingMore = true;

    try {
      const response = await getTaskFeed(
        slug,
        number,
        PAGE_SIZE,
        nextOffset,
      );
      if (requestGeneration !== generation) return;

      const seen = new Set(items.map(feedKey));
      const next = response.items.filter((item) => !seen.has(feedKey(item)));
      items = [...items, ...next];
      total = response.total;
      nextOffset =
        response.items.length > 0
          ? nextOffset + response.items.length
          : Math.max(nextOffset, total);
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : "Failed to load older activity",
      );
    } finally {
      if (requestGeneration === generation) {
        loadingMore = false;
      }
    }
  }

  async function handleAddComment(content: string) {
    await createComment(taskId, { content });
    void onCommentAdded().catch(() => {});
    await loadFirstPage(slug, number);
  }

  async function handleEditComment(commentId: number, content: string) {
    const updated = await updateComment(taskId, commentId, { content });
    items = items.map((item) =>
      item.type === "comment" && item.comment.id === commentId
        ? { ...item, comment: updated }
        : item,
    );
  }

  async function handleDeleteComment(commentId: number) {
    await deleteComment(taskId, commentId);

    const before = items.length;
    items = items.filter(
      (item) => !(item.type === "comment" && item.comment.id === commentId),
    );
    if (items.length < before) {
      total = Math.max(0, total - 1);
      nextOffset = Math.max(0, nextOffset - 1);
    }
  }

  function feedKey(item: FeedItem): string {
    return `${item.type}:${item.id}`;
  }

  function formatDateTime(d: string): string {
    return new Date(d).toLocaleString();
  }

  function initials(name: string): string {
    return name
      .split(/\s+/)
      .map((n) => n[0])
      .join("")
      .toUpperCase()
      .slice(0, 2);
  }

  function fieldLabel(field: string): string {
    return FIELD_LABELS[field] ?? field;
  }

  function formatChangeValue(change: HistoryChange): string {
    const value = change.new_value;
    return formatValue(change.field, value);
  }

  function formatOldValue(change: HistoryChange): string {
    return formatValue(change.field, change.old_value);
  }

  function formatValue(field: string, value: string | null): string {
    if (value === null) return "none";
    if (value === "") return "empty";

    if (field === "due_date") {
      const date = new Date(`${value}T00:00:00`);
      if (!Number.isNaN(date.getTime())) {
        return date.toLocaleDateString();
      }
    }

    return value;
  }
</script>

<div class="space-y-4">
  <CommentForm {taskId} onSubmit={handleAddComment} />

  {#if error && items.length > 0}
    <p class="text-destructive py-4 text-center text-sm">{error}</p>
  {/if}

  {#if loading && items.length === 0}
    <p class="text-muted-foreground py-4 text-center text-sm">
      Loading activity...
    </p>
  {:else if error && items.length === 0}
    <p class="text-destructive py-4 text-center text-sm">{error}</p>
  {:else if items.length === 0}
    <p class="text-muted-foreground text-sm italic">No activity yet</p>
  {:else}
    <div class="space-y-3">
      {#each items as item (feedKey(item))}
        {#if item.type === "comment"}
          <CommentItem
            comment={item.comment}
            {currentUserId}
            {taskId}
            {attachmentsMap}
            {autoLinkCtx}
            onEdit={handleEditComment}
            onDelete={handleDeleteComment}
          />
        {:else}
          <div class="flex gap-3 rounded-lg border bg-muted/30 p-3">
            <Avatar>
              <AvatarFallback
                >{initials(item.history.actor.name)}</AvatarFallback
              >
            </Avatar>
            <div class="min-w-0 flex-1">
              <div
                class="mb-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground"
              >
                <span class="font-medium text-foreground"
                  >{item.history.actor.name || "Unknown user"}</span
                >
                <span
                  >{item.history.action === "created"
                    ? "created this task"
                    : "updated the task"}</span
                >
                &middot;
                <span>{formatDateTime(item.history.created_at)}</span>
              </div>
              {#if item.history.action === "updated" && item.history.changes.length > 0}
                <ul class="space-y-1 text-sm">
                  {#each item.history.changes as change}
                    <li class="break-words">
                      <span class="text-muted-foreground"
                        >{fieldLabel(change.field)}:</span
                      >
                      <span class="text-muted-foreground whitespace-pre-wrap"
                        >{formatOldValue(change)}</span
                      >
                      <span class="text-muted-foreground">&rarr;</span>
                      <span class="whitespace-pre-wrap"
                        >{formatChangeValue(change)}</span
                      >
                    </li>
                  {/each}
                </ul>
              {/if}
            </div>
          </div>
        {/if}
      {/each}
    </div>

    {#if nextOffset < total}
      <div class="flex justify-center">
        <Button
          variant="outline"
          size="sm"
          onclick={loadMore}
          disabled={loadingMore}
        >
          {loadingMore
            ? "Loading..."
            : `Load older (${total - items.length} remaining)`}
        </Button>
      </div>
    {/if}
  {/if}
</div>
