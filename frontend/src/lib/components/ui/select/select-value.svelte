<script lang="ts">
  import * as SelectPrimitive from "bits-ui";

  let {
    children,
    placeholder,
    ...restProps
  }: SelectPrimitive.SelectValueProps = $props();
</script>

<SelectPrimitive.SelectValue {placeholder} {...restProps}>
  {@render children?.()}
</SelectPrimitive.SelectValue>
