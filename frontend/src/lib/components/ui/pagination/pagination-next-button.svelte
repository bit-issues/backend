<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as PaginationPrimitive from "bits-ui";
  import ChevronRight from "@lucide/svelte/icons/chevron-right";
  import { paginationButtonClass } from "./styles";

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: PaginationPrimitive.PaginationNextButtonProps = $props();
</script>

<PaginationPrimitive.PaginationNextButton
  bind:ref
  class={cn(paginationButtonClass, className)}
  {...restProps}
>
  <ChevronRight class="size-4" />
</PaginationPrimitive.PaginationNextButton>
