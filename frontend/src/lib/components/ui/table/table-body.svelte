<script lang="ts">
  import { cn } from "$lib/utils.js";
  import type { HTMLAttributes } from "svelte/elements";

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: HTMLAttributes<HTMLTableSectionElement> = $props();
</script>

<tbody
  bind:this={ref}
  class={cn("[&_tr:last-child]:border-0", className)}
  {...restProps}
>
  {@render children?.()}
</tbody>
