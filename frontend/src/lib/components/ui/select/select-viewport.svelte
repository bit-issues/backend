<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as SelectPrimitive from "bits-ui";

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: SelectPrimitive.SelectViewportProps = $props();
</script>

<SelectPrimitive.SelectViewport
  bind:ref
  class={cn("p-1", className)}
  {...restProps}
>
  {@render children?.()}
</SelectPrimitive.SelectViewport>
