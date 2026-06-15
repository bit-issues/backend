<script lang="ts">
  import { Textarea } from "$lib/components/ui/textarea";
  import { Button } from "$lib/components/ui/button";
  import { toast } from "$lib/toast";

  let {
    onSubmit = async (_content: string) => {},
  }: {
    onSubmit: (content: string) => Promise<void>;
  } = $props();

  let text = $state("");
  let posting = $state(false);

  async function handleSubmit() {
    if (!text.trim()) return;
    posting = true;
    try {
      await onSubmit(text.trim());
      text = "";
    } catch (e: any) {
      toast.error(e.message || "Failed to add comment");
    } finally {
      posting = false;
    }
  }
</script>

<div>
  <Textarea
    bind:value={text}
    placeholder="Add a comment..."
    class="mb-2 min-h-20"
  />
  <Button
    onclick={handleSubmit}
    disabled={posting || !text.trim()}
  >
    {posting ? "Posting..." : "Comment"}
  </Button>
</div>
