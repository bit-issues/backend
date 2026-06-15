<script lang="ts">
  import { cn, type WithElementRef } from "$lib/utils.js";

  type Props = WithElementRef<{
    value?: number;
    class?: string;
  }>;

  let {
    ref = $bindable(null),
    value = 0,
    class: className,
    ...restProps
  }: Props = $props();
</script>

<div
  bind:this={ref}
  role="progressbar"
  aria-valuemin={0}
  aria-valuemax={100}
  aria-valuenow={value}
  class={cn(
    "bg-muted relative h-2 w-full overflow-hidden rounded-full",
    className,
  )}
  {...restProps}
>
  <div
    class="bg-primary h-full w-full flex-1 transition-all duration-300"
    style="transform: translateX(-{100 - Math.max(0, Math.min(100, value))}%)"
  ></div>
</div>
