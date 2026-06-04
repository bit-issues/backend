<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as SelectPrimitive from "bits-ui";
  import Check from "@lucide/svelte/icons/check";

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: SelectPrimitive.SelectItemProps = $props();
</script>

<SelectPrimitive.SelectItem
  bind:ref
  class={cn(
    "relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-2 pr-8 text-sm outline-none focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
    className,
  )}
  {...restProps}
>
  <span class="absolute right-2 flex size-3.5 items-center justify-center">
    <SelectPrimitive.SelectItemIndicator>
      <Check class="size-4" />
    </SelectPrimitive.SelectItemIndicator>
  </span>
  <SelectPrimitive.SelectItemText>
    {@render children?.()}
  </SelectPrimitive.SelectItemText>
</SelectPrimitive.SelectItem>
