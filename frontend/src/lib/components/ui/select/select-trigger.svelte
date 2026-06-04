<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as SelectPrimitive from "bits-ui";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: SelectPrimitive.SelectTriggerProps = $props();
</script>

<SelectPrimitive.SelectTrigger
  bind:ref
  class={cn(
    "flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1",
    className,
  )}
  {...restProps}
>
  {@render children?.()}
  <ChevronDown class="size-4 opacity-50" />
</SelectPrimitive.SelectTrigger>
