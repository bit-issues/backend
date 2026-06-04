<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as PaginationPrimitive from "bits-ui";
  import { paginationButtonClass } from "./styles";

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: PaginationPrimitive.PaginationPageProps = $props();
</script>

<PaginationPrimitive.PaginationPage
  bind:ref
  class={cn(
    paginationButtonClass,
    "aria-[current=page]:bg-accent aria-[current=page]:text-accent-foreground",
    className,
  )}
  {...restProps}
/>
