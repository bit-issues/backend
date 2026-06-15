<script lang="ts">
  import { cn, type WithElementRef } from "$lib/utils.js";

  type Props = WithElementRef<{
    class?: string;
    children?: import("svelte").Snippet;
  }>;

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: Props = $props();
</script>

<div
  bind:this={ref}
  class={cn(
    "relative flex h-8 w-8 shrink-0 overflow-hidden rounded-full",
    className,
  )}
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</div>
