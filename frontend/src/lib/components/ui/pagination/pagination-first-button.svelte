<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as PaginationPrimitive from "bits-ui";
  import ChevronsLeft from "@lucide/svelte/icons/chevrons-left";
  import { paginationButtonClass } from "./styles";

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: PaginationPrimitive.PaginationFirstButtonProps = $props();
</script>

<PaginationPrimitive.PaginationFirstButton
  bind:ref
  class={cn(paginationButtonClass, className)}
  {...restProps}
>
  <ChevronsLeft class="size-4" />
</PaginationPrimitive.PaginationFirstButton>
