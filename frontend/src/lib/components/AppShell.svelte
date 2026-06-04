<script lang="ts">
  import type { Snippet } from "svelte";
  import Sidebar from "./Sidebar.svelte";

  let {
    currentPath = "/",
    children,
  }: {
    currentPath?: string;
    children: Snippet;
  } = $props();
</script>

<div class="flex min-h-screen">
  <Sidebar {currentPath} />
  <main class="flex-1 overflow-auto p-4 lg:p-6">
    {@render children()}
  </main>
</div>
