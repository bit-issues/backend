<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as SeparatorPrimitive from "bits-ui";

  let {
    ref = $bindable(null),
    class: className,
    orientation = "horizontal",
    decorative = true,
    ...restProps
  }: SeparatorPrimitive.Separator.RootProps = $props();
</script>

<SeparatorPrimitive.Separator.Root
  bind:ref
  {orientation}
  {decorative}
  class={cn(
    "shrink-0 bg-border data-[orientation=horizontal]:h-px data-[orientation=horizontal]:w-full data-[orientation=vertical]:h-full data-[orientation=vertical]:w-px",
    className,
  )}
  {...restProps}
/>
