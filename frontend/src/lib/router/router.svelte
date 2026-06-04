<script module lang="ts">
  function match(pattern: string, path: string): boolean {
    if (pattern === path) return true;
    const patParts = pattern.split("/");
    const pathParts = path.split("/");
    if (patParts.length !== pathParts.length) return false;
    return patParts.every((p, i) => p.startsWith(":") || p === pathParts[i]);
  }
</script>

<script lang="ts">
  import type { Snippet } from "svelte";

  let { routes }: { routes: Record<string, Snippet> } = $props();

  let path = $state(window.location.hash.slice(1) || "/");

  $effect(() => {
    const handler = () => {
      path = window.location.hash.slice(1) || "/";
    };
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  });
</script>

{#each Object.entries(routes) as [pattern, component]}
  {#if match(pattern, path)}
    {@render component()}
  {/if}
{/each}
