<script lang="ts">
  import { initUpload, confirmUpload } from "$lib/api/attachments";
  import { Progress } from "$lib/components/ui/progress";
  import { toast } from "$lib/toast";

  let {
    taskId = 0,
    onUploaded = (_att: any) => {},
  }: {
    taskId: number;
    onUploaded: (att: any) => void;
  } = $props();

  let dragging = $state(false);
  let uploading = $state(false);
  let progress = $state(0);

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    dragging = true;
  }

  function handleDragLeave() {
    dragging = false;
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragging = false;
    const file = e.dataTransfer?.files?.[0];
    if (file) upload(file);
  }

  function handleFilePick(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (file) upload(file);
    input.value = "";
  }

  async function upload(file: File) {
    if (uploading) return;

    const MAX_SIZE = 100 * 1024 * 1024;
    if (file.size > MAX_SIZE) {
      toast.error("File exceeds 100 MB limit");
      return;
    }

    uploading = true;
    progress = 0;

    try {
      const init = await initUpload(taskId, {
        file_name: file.name,
        size_bytes: file.size,
      });

      await new Promise<void>((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.timeout = 60_000;
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            progress = Math.round((e.loaded / e.total) * 100);
          }
        };
        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) resolve();
          else reject(new Error(`Upload failed with status ${xhr.status}`));
        };
        xhr.onerror = () =>
          reject(
            new Error("Upload failed: S3 storage not available in dev mode"),
          );
        xhr.ontimeout = () => reject(new Error("Upload timed out"));
        xhr.onabort = () => reject(new Error("Upload was canceled"));
        xhr.open("PUT", init.upload_url);
        xhr.send(file);
      });

      const confirmed = await confirmUpload(taskId, init.id);
      onUploaded(confirmed);
      toast.success("File uploaded");
    } catch (e: any) {
      toast.error(e.message || "Upload failed");
    } finally {
      uploading = false;
      progress = 0;
    }
  }
</script>

<div
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
  class="relative"
>
  {#if uploading}
    <div
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed px-3 py-4"
    >
      <span class="text-muted-foreground text-sm">Uploading...</span>
      <Progress value={progress} class="w-full" />
      <span class="text-muted-foreground text-xs">{progress}%</span>
    </div>
  {:else}
    <label
      class="flex cursor-pointer items-center justify-center rounded-lg border px-2.5 py-2 text-sm font-medium text-foreground hover:bg-muted"
      class:border-primary={dragging}
      class:bg-muted={dragging}
    >
      {#if dragging}
        Drop file here
      {:else}
        Upload file
      {/if}
      <input type="file" class="hidden" onchange={handleFilePick} />
    </label>
  {/if}
</div>
