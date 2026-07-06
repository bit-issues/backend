<script lang="ts">
  import { getTask } from "$lib/api/tasks";
  import { navigate } from "$lib/router/routes";

  let { params = {} }: { params?: Record<string, string> } = $props();

  let error = $state("");

  $effect(() => {
    let cancelled = false;
    const id = Number(params.id);
    if (!id) {
      error = "Invalid task ID";
      return;
    }

    getTask(id)
      .then((task) => {
        if (!cancelled) {
          navigate(`/tasks/${task.project_slug}/${task.number}`);
        }
      })
      .catch((e: any) => {
        if (!cancelled) {
          error = e.message || "Failed to load task";
        }
      });

    return () => {
      cancelled = true;
    };
  });
</script>

<div class="flex min-h-[50vh] items-center justify-center">
  {#if error}
    <p class="text-destructive text-sm">{error}</p>
  {:else}
    <p class="text-muted-foreground text-sm">Redirecting...</p>
  {/if}
</div>
