<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as PaginationPrimitive from "bits-ui";
  import ChevronsRight from "@lucide/svelte/icons/chevrons-right";
  import { paginationButtonClass } from "./styles";

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: PaginationPrimitive.PaginationLastButtonProps = $props();
</script>

<PaginationPrimitive.PaginationLastButton
  bind:ref
  class={cn(paginationButtonClass, className)}
  {...restProps}
>
  <ChevronsRight class="size-4" />
</PaginationPrimitive.PaginationLastButton>
