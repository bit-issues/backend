<script lang="ts">
  import { getUser } from "$lib/stores/auth.svelte";
  import * as Card from "$lib/components/ui/card";
  import * as Badge from "$lib/components/ui/badge";
  import { navigate } from "$lib/router/routes";

  let user = $derived(getUser());

  const statusBadgeColors: Record<string, string> = {
    active: "border-transparent bg-green-100 text-green-700 dark:bg-green-300/15 dark:text-green-300",
    pending: "border-transparent bg-amber-100 text-amber-700 dark:bg-amber-300/15 dark:text-amber-300",
    blocked: "border-transparent bg-red-100 text-red-700 dark:bg-red-300/15 dark:text-red-300",
  };

  const roleBadgeColors: Record<string, string> = {
    admin: "border-transparent bg-purple-100 text-purple-700 dark:bg-purple-300/15 dark:text-purple-300",
    user: "border-transparent bg-blue-100 text-blue-700 dark:bg-blue-300/15 dark:text-blue-300",
  };

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString();
  }

</script>

<div class="mx-auto max-w-2xl py-8">
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-2xl font-bold">Profile</h1>
    <button
      type="button"
      onclick={() => navigate("/settings/security")}
      class="text-primary text-sm underline-offset-4 hover:underline cursor-pointer"
    >
      Security settings
    </button>
  </div>

  <Card.Root>
    <Card.Header>
      <Card.Title>Account Info</Card.Title>
      <Card.Description>Your account details</Card.Description>
    </Card.Header>
    <Card.Content>
      {#if user}
        <dl class="flex flex-col gap-3">
          <div class="flex flex-col gap-0.5">
            <dt class="text-muted-foreground text-xs font-medium">ID</dt>
            <dd class="text-sm">{user.id}</dd>
          </div>
          <div class="flex flex-col gap-0.5">
            <dt class="text-muted-foreground text-xs font-medium">Name</dt>
            <dd class="text-sm font-medium">{user.name}</dd>
          </div>
          <div class="flex flex-col gap-0.5">
            <dt class="text-muted-foreground text-xs font-medium">Email</dt>
            <dd class="text-sm">{user.email}</dd>
          </div>
          <div class="flex flex-col gap-0.5">
            <dt class="text-muted-foreground text-xs font-medium">Role</dt>
            <dd>
              <Badge.Root class={roleBadgeColors[user.role]}>
                {user.role}
              </Badge.Root>
            </dd>
          </div>
          <div class="flex flex-col gap-0.5">
            <dt class="text-muted-foreground text-xs font-medium">Status</dt>
            <dd>
              <Badge.Root class={statusBadgeColors[user.status]}>
                {user.status}
              </Badge.Root>
            </dd>
          </div>
          <div class="flex flex-col gap-0.5">
            <dt class="text-muted-foreground text-xs font-medium">Member since</dt>
            <dd class="text-sm">{formatDate(user.created_at)}</dd>
          </div>
        </dl>
      {:else}
        <p class="text-muted-foreground text-sm">Not available</p>
      {/if}
    </Card.Content>
  </Card.Root>
</div>
