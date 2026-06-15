<script lang="ts">
  import type { Attachment } from "$lib/types/api";
  import { toast } from "$lib/toast";

  let {
    attachments = [] as Attachment[],
    currentUserId = 0,
    onDelete = async (_attachmentId: number) => {},
  }: {
    attachments: Attachment[];
    currentUserId: number;
    onDelete: (attachmentId: number) => Promise<void>;
  } = $props();

  function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }
</script>

{#if attachments.length > 0}
  <div class="mb-3 space-y-1">
    {#each attachments as att (att.id)}
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
        {#if att.uploaded_by.id === currentUserId}
          <button
            type="button"
            onclick={async () => {
              if (confirm("Delete this attachment?")) {
                try {
                  await onDelete(att.id);
                } catch (e: any) {
                  toast.error(e.message || "Failed to delete attachment");
                }
              }
            }}
            class="text-muted-foreground hover:text-destructive shrink-0 cursor-pointer text-xs underline underline-offset-2"
          >
            Delete
          </button>
        {/if}
      </div>
    {/each}
  </div>
{:else}
  <p class="text-muted-foreground mb-3 text-sm italic">No attachments</p>
{/if}
