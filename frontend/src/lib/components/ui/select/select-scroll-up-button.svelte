<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as SelectPrimitive from "bits-ui";
  import ChevronUp from "@lucide/svelte/icons/chevron-up";

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: SelectPrimitive.SelectScrollUpButtonProps = $props();
</script>

<SelectPrimitive.SelectScrollUpButton
  bind:ref
  class={cn("flex cursor-default items-center justify-center py-1", className)}
  {...restProps}
>
  <ChevronUp class="size-4" />
</SelectPrimitive.SelectScrollUpButton>
