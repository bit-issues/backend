<script lang="ts">
  import type { RouteDef } from "./routes";
  import { matchRoute } from "./routes";
  import Guard from "./guard.svelte";

  let {
    routes,
    notFound: NotFoundPage,
  }: {
    routes: RouteDef[];
    notFound?: import("svelte").ComponentType;
  } = $props();

  function hashPath(): string {
    return (window.location.hash.slice(1) || "/").split("?")[0];
  }

  let path = $state(hashPath());

  $effect(() => {
    const handler = () => {
      path = hashPath();
    };
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  });

  let match = $derived(matchRoute(path, routes));
</script>

{#if match}
  {@const Comp = match.route.component}
  <Guard auth={match.route.auth} role={match.route.role}>
    <Comp params={match.params} />
  </Guard>
{:else if NotFoundPage}
  <NotFoundPage params={{}} />
{/if}
