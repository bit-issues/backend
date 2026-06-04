<script lang="ts">
  import * as Card from "$lib/components/ui/card";
  import { getUser } from "$lib/stores/auth.svelte";
  import BugIcon from "@lucide/svelte/icons/bug";

  let { params = {} }: { params?: Record<string, string> } = $props();
</script>

<div class="mx-auto max-w-2xl space-y-6">
  <div class="flex items-center gap-3">
    <BugIcon class="size-8" />
    <div>
      <h1 class="text-2xl font-semibold">
        Welcome, {getUser()?.name || "User"}
      </h1>
      <p class="text-muted-foreground text-sm">
        BitIssues — task tracking, simplified.
      </p>
    </div>
  </div>

  <div class="grid gap-4 sm:grid-cols-2">
    <Card.Root>
      <Card.CardHeader>
        <Card.CardTitle>My tasks</Card.CardTitle>
        <Card.CardDescription>Tasks assigned to you</Card.CardDescription>
      </Card.CardHeader>
      <Card.CardContent>
        <p class="text-muted-foreground text-sm">Coming soon in Phase 2</p>
      </Card.CardContent>
    </Card.Root>
    <Card.Root>
      <Card.CardHeader>
        <Card.CardTitle>Recent projects</Card.CardTitle>
        <Card.CardDescription>Your active projects</Card.CardDescription>
      </Card.CardHeader>
      <Card.CardContent>
        <p class="text-muted-foreground text-sm">Coming soon in Phase 2</p>
      </Card.CardContent>
    </Card.Root>
  </div>
</div>
