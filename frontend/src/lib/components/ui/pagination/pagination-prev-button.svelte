<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as PaginationPrimitive from "bits-ui";
  import ChevronLeft from "@lucide/svelte/icons/chevron-left";
  import { paginationButtonClass } from "./styles.js";

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: PaginationPrimitive.PaginationPrevButtonProps = $props();
</script>

<PaginationPrimitive.PaginationPrevButton
  bind:ref
  class={cn(paginationButtonClass, className)}
  {...restProps}
>
  <ChevronLeft class="size-4" />
</PaginationPrimitive.PaginationPrevButton>
