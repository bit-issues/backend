<script lang="ts">
  import { Avatar, AvatarFallback } from "$lib/components/ui/avatar";
  import { Button } from "$lib/components/ui/button";
  import { Textarea } from "$lib/components/ui/textarea";
  import InlineImageButton from "$lib/components/InlineImageButton.svelte";
  import { parse } from "marked";
  import { processAutoLinks } from "$lib/autolink";
  import { resolveAttachmentRefs } from "$lib/resolve-attachment-refs";
  import { InlineImageEditor } from "$lib/inline-image-editor.svelte";
  import DOMPurify from "dompurify";
  import type { AutoLinkContext } from "$lib/autolink";
  import type { Comment } from "$lib/types/api";
  import { isAdmin } from "$lib/stores/auth.svelte";
  import { toast } from "$lib/toast";

  let {
    comments = [] as Comment[],
    currentUserId = 0,
    taskId = 0,
    attachmentsMap = new Map<number, string>(),
    autoLinkCtx = {} as AutoLinkContext,
    onEdit = async (_commentId: number, _content: string) => {},
    onDelete = async (_commentId: number) => {},
  }: {
    comments: Comment[];
    currentUserId: number;
    taskId?: number;
    attachmentsMap: Map<number, string>;
    autoLinkCtx: AutoLinkContext;
    onEdit: (commentId: number, content: string) => Promise<void>;
    onDelete: (commentId: number) => Promise<void>;
  } = $props();

  let editingId = $state<number | null>(null);
  let editText = $state("");
  let saving = $state(false);
  let editTextarea: HTMLTextAreaElement | undefined = $state();
  let fileInput: HTMLInputElement | undefined = $state();

  const images = new InlineImageEditor({
    getTaskId: () => taskId,
    getValue: () => editText,
    setValue: (v) => {
      editText = v;
    },
    getTextarea: () => editTextarea,
    getSessionId: () => editingId,
  });

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
      parse(
        processAutoLinks(
          resolveAttachmentRefs(text, attachmentsMap),
          autoLinkCtx,
        ),
        { async: false },
      ) as string,
    );
  }

  function startEdit(comment: Comment) {
    if (images.uploading) return;
    editingId = comment.id;
    editText = comment.content;
  }

  function cancelEdit() {
    if (images.uploading) return;
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
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to update comment");
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
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            ondragover={(e) => {
              if (saving) {
                e.preventDefault();
                return;
              }
              images.handleDragOver(e);
            }}
            ondrop={(e) => {
              if (saving) {
                e.preventDefault();
                return;
              }
              images.handleImageDrop(e);
            }}
          >
            <Textarea
              bind:value={editText}
              bind:this={editTextarea}
              onpaste={(e) => {
                if (!saving) images.handleImagePaste(e);
              }}
              class="mb-2 min-h-16"
            />
          </div>
          <div class="mb-2 flex items-center gap-2">
            <InlineImageButton
              uploading={images.uploading}
              disabled={!taskId || saving}
              onclick={() => fileInput?.click()}
            />
            <input
              type="file"
              disabled={saving || images.uploading}
              bind:this={fileInput}
              accept="image/png,image/jpeg,image/gif,image/webp"
              class="hidden"
              onchange={(e) => images.handleImagePick(e)}
            />
          </div>
          <div class="flex gap-2">
            <Button
              size="sm"
              onclick={() => handleEdit(comment.id)}
              disabled={saving || images.uploading}
            >
              {saving ? "Saving..." : "Save"}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onclick={cancelEdit}
              disabled={images.uploading}
            >
              Cancel
            </Button>
          </div>
        {:else}
          <div class="prose prose-sm dark:prose-invert max-w-none text-sm">
            {@html renderMarkdown(comment.content)}
          </div>
          {#if canManageComment(comment)}
            <div class="mt-1 flex gap-2">
              <button
                class="text-muted-foreground hover:text-foreground cursor-pointer text-xs underline underline-offset-2 disabled:opacity-50 disabled:no-underline"
                disabled={images.uploading}
                onclick={() => startEdit(comment)}>Edit</button
              >
              <button
                class="text-muted-foreground hover:text-destructive cursor-pointer text-xs underline underline-offset-2 disabled:opacity-50 disabled:no-underline"
                disabled={images.uploading}
                onclick={async () => {
                  if (confirm("Delete this comment?")) {
                    try {
                      await onDelete(comment.id);
                    } catch (e) {
                      toast.error(
                        e instanceof Error
                          ? e.message
                          : "Failed to delete comment",
                      );
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
