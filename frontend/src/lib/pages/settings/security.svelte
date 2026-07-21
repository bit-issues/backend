<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import {
    passkeyRegisterBegin,
    passkeyRegisterComplete,
    listPasskeys,
    renamePasskey,
    deletePasskey,
  } from "$lib/api/passkey";
  import { changePasswordApi } from "$lib/api/auth";
  import { toast } from "$lib/toast";
  import type { PasskeyCredential } from "$lib/types/api";
  import KeyIcon from "@lucide/svelte/icons/key";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import PencilIcon from "@lucide/svelte/icons/pencil";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import CheckIcon from "@lucide/svelte/icons/check";
  import XIcon from "@lucide/svelte/icons/x";

  let credentials = $state<PasskeyCredential[]>([]);
  let loading = $state(true);
  let registering = $state(false);
  let editingId = $state<number | null>(null);
  let editName = $state("");
  let passkeySupported = $state(false);

  let oldPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let saving = $state(false);
  let error = $state("");

  let passwordsMatch = $derived(newPassword === confirmPassword);
  let passwordValid = $derived(newPassword.length >= 8);
  let canSubmit = $derived(
    !!oldPassword &&
      !!newPassword &&
      !!confirmPassword &&
      passwordValid &&
      passwordsMatch,
  );

  $effect(() => {
    passkeySupported =
      typeof navigator !== "undefined" &&
      "credentials" in navigator &&
      "PublicKeyCredential" in window;
  });

  async function load() {
    loading = true;
    try {
      credentials = await listPasskeys();
    } catch (e: any) {
      toast.error(e?.message || "Failed to load passkeys");
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    load();
  });

  async function handleRegister() {
    if (registering) return;
    registering = true;

    try {
      const options = await passkeyRegisterBegin();
      const credential = await navigator.credentials.create({
        publicKey: options,
      });
      if (!credential) throw new Error("Passkey creation cancelled");

      await passkeyRegisterComplete(credential as PublicKeyCredential);
      toast.success("Passkey registered successfully");
      await load();
    } catch (e: any) {
      toast.error(e?.message || "Failed to register passkey");
    } finally {
      registering = false;
    }
  }

  function startEdit(cred: PasskeyCredential) {
    editingId = cred.id;
    editName = cred.name;
  }

  function cancelEdit() {
    editingId = null;
    editName = "";
  }

  async function saveEdit(cred: PasskeyCredential) {
    if (!editName.trim()) return;
    try {
      await renamePasskey(cred.id, { name: editName.trim() });
      toast.success("Passkey renamed");
      editingId = null;
      editName = "";
      await load();
    } catch (e: any) {
      toast.error(e?.message || "Failed to rename passkey");
    }
  }

  async function handleDelete(cred: PasskeyCredential) {
    if (!confirm(`Delete passkey "${cred.name}"?`)) return;
    try {
      await deletePasskey(cred.id);
      toast.success("Passkey deleted");
      await load();
    } catch (e: any) {
      toast.error(e?.message || "Failed to delete passkey");
    }
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString();
  }

  async function handleSubmit() {
    if (!canSubmit || saving) return;
    saving = true;
    error = "";
    try {
      await changePasswordApi({
        old_password: oldPassword,
        new_password: newPassword,
      });
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
  <div class="mb-6 flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-bold">Security</h1>
      <p class="text-muted-foreground text-sm mt-1">
        Manage your passkeys and security settings
      </p>
    </div>
    {#if passkeySupported}
      <Button onclick={handleRegister} disabled={registering}>
        <PlusIcon class="mr-2 size-4" />
        {registering ? "Registering..." : "Add passkey"}
      </Button>
    {/if}
  </div>

  <div class="flex flex-col gap-6">
    <Card.Root>
      <Card.Header>
        <Card.Title>Change Password</Card.Title>
        <Card.Description>Update your account password</Card.Description>
      </Card.Header>
      <Card.Content>
        <form
          onsubmit={(e) => {
            e.preventDefault();
            handleSubmit();
          }}
          class="flex flex-col gap-4"
        >
          <div class="flex flex-col gap-2">
            <Label for="old_password">Current Password</Label>
            <Input
              id="old_password"
              type="password"
              autocomplete="current-password"
              bind:value={oldPassword}
              oninput={() => (error = "")}
              disabled={saving}
              placeholder="Enter current password"
            />
          </div>
          <div class="flex flex-col gap-2">
            <Label for="new_password">New Password</Label>
            <Input
              id="new_password"
              type="password"
              autocomplete="new-password"
              bind:value={newPassword}
              disabled={saving}
              placeholder="Enter new password"
            />
            {#if newPassword && !passwordValid}
              <p class="text-destructive text-xs">
                Password must be at least 8 characters
              </p>
            {/if}
          </div>
          <div class="flex flex-col gap-2">
            <Label for="confirm_password">Confirm New Password</Label>
            <Input
              id="confirm_password"
              type="password"
              autocomplete="new-password"
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
          {#if error}
            <p class="text-destructive text-xs">{error}</p>
          {/if}
        </form>
      </Card.Content>
    </Card.Root>

    <Card.Root>
      <Card.Header>
        <Card.Title>Passkeys</Card.Title>
        <Card.Description>
          Passkeys let you sign in without a password using biometrics, PIN, or
          security keys.
        </Card.Description>
      </Card.Header>
      <Card.Content>
        {#if loading}
          <p class="text-muted-foreground text-sm">Loading...</p>
        {:else if credentials.length === 0}
          <div class="flex flex-col items-center gap-2 py-8 text-center">
            <KeyIcon class="text-muted-foreground size-8" />
            <p class="text-muted-foreground text-sm">
              {passkeySupported
                ? "No passkeys registered yet."
                : "Passkeys are not supported in this browser."}
            </p>
          </div>
        {:else}
          <ul class="flex flex-col gap-2">
            {#each credentials as cred (cred.id)}
              <li
                class="flex items-center justify-between rounded-lg border border-border p-3"
              >
                {#if editingId === cred.id}
                  <div class="flex flex-1 items-center gap-2">
                    <Input
                      bind:value={editName}
                      class="h-8"
                      placeholder="Passkey name"
                      onkeydown={(e) => {
                        if (e.key === "Enter") saveEdit(cred);
                        if (e.key === "Escape") cancelEdit();
                      }}
                    />
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label="Save passkey name"
                      onclick={() => saveEdit(cred)}
                    >
                      <CheckIcon class="size-4" />
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label="Cancel rename"
                      onclick={cancelEdit}
                    >
                      <XIcon class="size-4" />
                    </Button>
                  </div>
                {:else}
                  <div class="flex flex-1 items-center gap-3">
                    <KeyIcon class="text-muted-foreground size-4" />
                    <div>
                      <p class="text-sm font-medium">{cred.name}</p>
                      <p class="text-muted-foreground text-xs">
                        Registered {formatDate(cred.created_at)}
                      </p>
                    </div>
                  </div>
                  <div class="flex items-center gap-1">
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label="Rename passkey"
                      onclick={() => startEdit(cred)}
                    >
                      <PencilIcon class="size-4" />
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      aria-label="Delete passkey"
                      class="text-destructive hover:text-destructive"
                      onclick={() => handleDelete(cred)}
                    >
                      <Trash2Icon class="size-4" />
                    </Button>
                  </div>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </Card.Content>
    </Card.Root>
  </div>
</div>
