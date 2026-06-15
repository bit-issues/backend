<script lang="ts">
  import { cn, type WithElementRef } from "$lib/utils.js";

  type Props = WithElementRef<{
    src: string;
    alt?: string;
    class?: string;
  }>;

  let {
    ref = $bindable(null),
    src,
    alt = "",
    class: className,
    ...restProps
  }: Props = $props();

  let error = $state(false);

  $effect(() => {
    // Reset load failure state whenever a new source is provided
    if (src) error = false;
  });
</script>

{#if src && !error}
  <img
    bind:this={ref}
    {src}
    {alt}
    onerror={() => (error = true)}
    class={cn("aspect-square h-full w-full", className)}
    {...restProps}
  />
{/if}
