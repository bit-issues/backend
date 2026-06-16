<script lang="ts">
  import { onMount } from "svelte";
  import { navigate } from "$lib/router/routes";
  onMount(() => navigate("/admin/users"));
</script>
