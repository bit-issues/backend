<script lang="ts">
  import { Avatar, AvatarFallback } from "$lib/components/ui/avatar";
  import { Button } from "$lib/components/ui/button";
  import { Textarea } from "$lib/components/ui/textarea";
  import { parse } from "marked";
  import { processAutoLinks } from "$lib/autolink";
  import DOMPurify from "dompurify";
  import type { AutoLinkContext } from "$lib/autolink";
  import type { Comment } from "$lib/types/api";
  import { isAdmin } from "$lib/stores/auth.svelte";
  import { toast } from "$lib/toast";

  let {
    comments = [] as Comment[],
    currentUserId = 0,
    autoLinkCtx = {} as AutoLinkContext,
    onEdit = async (_commentId: number, _content: string) => {},
    onDelete = async (_commentId: number) => {},
  }: {
    comments: Comment[];
    currentUserId: number;
    autoLinkCtx: AutoLinkContext;
    onEdit: (commentId: number, content: string) => Promise<void>;
    onDelete: (commentId: number) => Promise<void>;
  } = $props();

  let editingId = $state<number | null>(null);
  let editText = $state("");
  let saving = $state(false);

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

  function renderMarkdown(text: string): string {
    if (!text) return "";
    return DOMPurify.sanitize(
      parse(processAutoLinks(text, autoLinkCtx), {
        async: false,
      }) as string,
    );
  }

  function startEdit(comment: Comment) {
    editingId = comment.id;
    editText = comment.content;
  }

  function cancelEdit() {
    editingId = null;
    editText = "";
  }

  function canManageComment(comment: Comment): boolean {
    return isAdmin() || comment.author.id === currentUserId;
  }

  async function handleEdit(commentId: number) {
    if (!editText.trim()) return;
    saving = true;
    try {
      await onEdit(commentId, editText);
      editingId = null;
      editText = "";
    } catch (e: any) {
      toast.error(e.message || "Failed to update comment");
    } finally {
      saving = false;
    }
  }
</script>

<div class="space-y-3">
  {#each comments as comment (comment.id)}
    <div class="flex gap-3 rounded-lg border p-3">
      <Avatar>
        <AvatarFallback>{initials(comment.author.name)}</AvatarFallback>
      </Avatar>
      <div class="min-w-0 flex-1">
        <div class="mb-1 flex items-center gap-2 text-xs text-muted-foreground">
          <span class="font-medium text-foreground">{comment.author.name}</span>
          &middot;
          <span>{formatDateTime(comment.created_at)}</span>
          {#if comment.updated_at !== comment.created_at}
            &middot;
            <span class="italic">edited</span>
          {/if}
        </div>

        {#if editingId === comment.id}
          <Textarea bind:value={editText} class="mb-2 min-h-16" />
          <div class="flex gap-2">
            <Button
              size="sm"
              onclick={() => handleEdit(comment.id)}
              disabled={saving}
            >
              {saving ? "Saving..." : "Save"}
            </Button>
            <Button size="sm" variant="ghost" onclick={cancelEdit}
              >Cancel</Button
            >
          </div>
        {:else}
          <div class="prose prose-sm dark:prose-invert max-w-none text-sm">
            {@html renderMarkdown(comment.content)}
          </div>
          {#if canManageComment(comment)}
            <div class="mt-1 flex gap-2">
              <button
                class="text-muted-foreground hover:text-foreground cursor-pointer text-xs underline underline-offset-2"
                onclick={() => startEdit(comment)}>Edit</button
              >
              <button
                class="text-muted-foreground hover:text-destructive cursor-pointer text-xs underline underline-offset-2"
                onclick={async () => {
                  if (confirm("Delete this comment?")) {
                    try {
                      await onDelete(comment.id);
                    } catch (e: any) {
                      toast.error(e.message || "Failed to delete comment");
                    }
                  }
                }}>Delete</button
              >
            </div>
          {/if}
        {/if}
      </div>
    </div>
  {:else}
    <p class="text-muted-foreground text-sm italic">No comments yet</p>
  {/each}
</div>
