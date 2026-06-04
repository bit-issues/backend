<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as SelectPrimitive from "bits-ui";

  let {
    ref = $bindable(null),
    class: className,
    ...restProps
  }: SelectPrimitive.SelectSeparatorProps = $props();
</script>

<SelectPrimitive.SelectSeparator
  bind:ref
  class={cn("-mx-1 my-1 h-px bg-muted", className)}
  {...restProps}
/>
