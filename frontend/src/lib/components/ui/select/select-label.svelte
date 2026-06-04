<script lang="ts">
  import { cn } from "$lib/utils.js";
  import * as SelectPrimitive from "bits-ui";

  let {
    ref = $bindable(null),
    class: className,
    children,
    ...restProps
  }: SelectPrimitive.SelectLabelProps = $props();
</script>

<SelectPrimitive.SelectLabel
  bind:ref
  class={cn("px-2 py-1.5 text-sm font-semibold", className)}
  {...restProps}
>
  {@render children?.()}
</SelectPrimitive.SelectLabel>
