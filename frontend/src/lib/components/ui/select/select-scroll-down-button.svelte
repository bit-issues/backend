<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as SelectPrimitive from "bits-ui";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: SelectPrimitive.SelectScrollDownButtonProps = $props();
</script>

<SelectPrimitive.SelectScrollDownButton
  bind:ref
  class={cn("flex cursor-default items-center justify-center py-1", className)}
  {...restProps}
>
  <ChevronDown class="size-4" />
</SelectPrimitive.SelectScrollDownButton>
