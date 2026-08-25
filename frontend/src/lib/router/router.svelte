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
    const hash = window.location.hash.slice(1);
    // The Bitbucket OAuth callback redirects to /admin?oauth=... without a
    // hash; treat that pathname landing as the /admin route.
    return (hash || (window.location.pathname === "/admin" ? "/admin" : "/")).split("?")[0];
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
