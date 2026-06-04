<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as PaginationPrimitive from "bits-ui";

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: PaginationPrimitive.PaginationListProps = $props();
</script>

<PaginationPrimitive.PaginationList
  bind:ref
  class={cn("flex flex-row items-center gap-1", className)}
  {...restProps}
>
  {@render children?.()}
</PaginationPrimitive.PaginationList>
