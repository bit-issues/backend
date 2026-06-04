<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as PaginationPrimitive from "bits-ui";
  import { type ComponentProps, type Snippet } from "svelte";

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: PaginationPrimitive.PaginationProps = $props();
</script>

<PaginationPrimitive.Pagination
  bind:ref
  class={cn("mx-auto flex w-full justify-center", className)}
  {...restProps}
>
  {@render children?.()}
</PaginationPrimitive.Pagination>
