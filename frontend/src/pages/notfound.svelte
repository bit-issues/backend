<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { navigate } from "$lib/router/routes";
  import { isAuthenticated } from "$lib/stores/auth.svelte";

  let { params = {} }: { params?: Record<string, string> } = $props();
</script>

<div class="flex min-h-[60vh] items-center justify-center p-4">
  <Card.Root size="sm" class="w-full max-w-sm text-center">
    <Card.CardHeader>
      <Card.CardTitle class="text-4xl">404</Card.CardTitle>
      <Card.CardDescription>Page not found</Card.CardDescription>
    </Card.CardHeader>
    <Card.CardContent>
      <Button
        onclick={() => navigate(isAuthenticated() ? "/" : "/login")}
        class="w-full"
      >
        Go home
      </Button>
    </Card.CardContent>
  </Card.Root>
</div>
