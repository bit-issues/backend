<script lang="ts">
  import * as SelectPrimitive from "bits-ui";

  let { children, ...restProps }: SelectPrimitive.SelectGroupProps = $props();
</script>

<SelectPrimitive.SelectGroup {...restProps}>
  {@render children?.()}
</SelectPrimitive.SelectGroup>
