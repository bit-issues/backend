<script lang="ts">
  import { changePasswordApi } from "$lib/api/auth";
  import { getUser } from "$lib/stores/auth.svelte";
  import { toast } from "$lib/toast";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import * as Badge from "$lib/components/ui/badge";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";

  let user = $derived(getUser());

  let oldPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let saving = $state(false);
  let error = $state("");

  let passwordsMatch = $derived(newPassword === confirmPassword);
  let passwordValid = $derived(newPassword.length >= 8);
  let canSubmit = $derived(!!oldPassword && !!newPassword && !!confirmPassword && passwordValid && passwordsMatch);

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

  async function handleSubmit() {
    if (!canSubmit || saving) return;
    saving = true;
    error = "";
    try {
      await changePasswordApi({ old_password: oldPassword, new_password: newPassword });
      toast.success("Password changed successfully");
      oldPassword = "";
      newPassword = "";
      confirmPassword = "";
    } catch (e: any) {
      error = e?.message || "Failed to change password";
      toast.error(error);
    } finally {
      saving = false;
    }
  }
</script>

<div class="mx-auto max-w-2xl py-8">
  <h1 class="mb-6 text-2xl font-bold">Profile</h1>

  <div class="grid gap-6 md:grid-cols-2">
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

    <Card.Root>
      <Card.Header>
        <Card.Title>Change Password</Card.Title>
        <Card.Description>Update your account password</Card.Description>
      </Card.Header>
      <Card.Content>
        <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="flex flex-col gap-4">
          <div class="flex flex-col gap-2">
            <Label for="old_password">Current Password</Label>
            <Input
              id="old_password"
              type="password"
              bind:value={oldPassword}
              disabled={saving}
              placeholder="Enter current password"
            />
          </div>
          <div class="flex flex-col gap-2">
            <Label for="new_password">New Password</Label>
            <Input
              id="new_password"
              type="password"
              bind:value={newPassword}
              disabled={saving}
              placeholder="Enter new password"
            />
            {#if newPassword && !passwordValid}
              <p class="text-destructive text-xs">Password must be at least 8 characters</p>
            {/if}
          </div>
          <div class="flex flex-col gap-2">
            <Label for="confirm_password">Confirm New Password</Label>
            <Input
              id="confirm_password"
              type="password"
              bind:value={confirmPassword}
              disabled={saving}
              placeholder="Confirm new password"
            />
            {#if confirmPassword && !passwordsMatch}
              <p class="text-destructive text-xs">Passwords do not match</p>
            {/if}
          </div>
          <Button type="submit" disabled={!canSubmit || saving} class="mt-2">
            {saving ? "Saving..." : "Change Password"}
          </Button>
        </form>
      </Card.Content>
    </Card.Root>
  </div>
</div>
