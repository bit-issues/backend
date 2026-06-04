<script lang="ts">
  import type { Snippet } from "svelte";
  import { isAuthenticated, isAdmin } from "$lib/stores/auth.svelte";
  import { navigate } from "./routes";

  let {
    auth = false,
    role,
    children,
  }: {
    auth?: boolean;
    role?: "admin";
    children: Snippet;
  } = $props();

  $effect(() => {
    if (auth && !isAuthenticated()) {
      navigate("/login");
    } else if (role === "admin" && !isAdmin()) {
      navigate("/");
    }
  });
</script>

{#if (!auth || isAuthenticated()) && (!role || isAdmin())}
  {@render children()}
{/if}
