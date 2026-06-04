<script lang="ts">
  import { navigate } from "$lib/router/routes";

  let { params = {} }: { params?: Record<string, string> } = $props();

  $effect(() => {
    navigate("/dashboard");
  });
</script>
