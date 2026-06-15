<script lang="ts">
  import { cn, type WithElementRef } from "$lib/utils.js";

  type Props = WithElementRef<{
    class?: string;
  }>;

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: Props = $props();
</script>

<span
  bind:this={ref}
  class={cn(
    "bg-muted flex h-full w-full items-center justify-center rounded-full text-xs font-medium",
    className,
  )}
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</span>
