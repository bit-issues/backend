<script lang="ts">
  import { Textarea } from "$lib/components/ui/textarea";
  import { Button } from "$lib/components/ui/button";
  import InlineImageButton from "$lib/components/InlineImageButton.svelte";
  import { InlineImageEditor } from "$lib/inline-image-editor.svelte";
  import { toast } from "$lib/toast";

  let {
    taskId = 0,
    onSubmit = async (_content: string) => {},
  }: {
    taskId?: number;
    onSubmit: (content: string) => Promise<void>;
  } = $props();

  let text = $state("");
  let posting = $state(false);
  let textareaEl: HTMLTextAreaElement | undefined = $state();
  let fileInput: HTMLInputElement | undefined = $state();

  const images = new InlineImageEditor({
    getTaskId: () => taskId,
    getValue: () => text,
    setValue: (v) => {
      text = v;
    },
    getTextarea: () => textareaEl,
  });

  async function handleSubmit() {
    if (!text.trim()) return;
    if (images.uploading) return;
    posting = true;
    try {
      await onSubmit(text.trim());
      text = "";
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to add comment");
    } finally {
      posting = false;
    }
  }
</script>

<div>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    ondragover={(e) => images.handleDragOver(e)}
    ondrop={(e) => images.handleImageDrop(e)}
  >
    <Textarea
      bind:value={text}
      bind:this={textareaEl}
      onpaste={(e) => images.handleImagePaste(e)}
      placeholder="Add a comment..."
      class="mb-2 min-h-20"
    />
  </div>
  <div class="mb-2 flex items-center gap-2">
    <InlineImageButton
      uploading={images.uploading}
      disabled={!taskId}
      onclick={() => fileInput?.click()}
    />
    <input
      type="file"
      bind:this={fileInput}
      accept="image/png,image/jpeg,image/gif,image/webp"
      class="hidden"
      onchange={(e) => images.handleImagePick(e)}
    />
  </div>
  <Button onclick={handleSubmit} disabled={posting || images.uploading || !text.trim()}>
    {posting ? "Posting..." : "Comment"}
  </Button>
</div>
