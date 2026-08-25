<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import * as Badge from "$lib/components/ui/badge";
  import * as Dialog from "$lib/components/ui/dialog";
  import {
    getProjectWebhookStatus,
    registerProjectWebhook,
    unregisterProjectWebhook,
  } from "$lib/api/webhooks";
  import { toast } from "$lib/toast";
  import { navigate } from "$lib/router/routes";
  import type { ProjectWebhookStatus } from "$lib/types/api";

  let { slug }: { slug: string } = $props();

  let status = $state<ProjectWebhookStatus | null>(null);
  let loading = $state(true);
  let loadError = $state("");
  let errorCode = $state<number | null>(null);
  let busy = $state(false);
  let showUnregisterDialog = $state(false);

  let oauthRevoked = $derived(errorCode === 401);
  let permissionDenied = $derived(errorCode === 403);

  const badgeColors: Record<string, string> = {
    registered:
      "border-transparent bg-green-100 text-green-700 dark:bg-green-300/15 dark:text-green-300",
    not_registered:
      "border-transparent bg-gray-100 text-gray-600 dark:bg-gray-300/15 dark:text-gray-300",
    failed:
      "border-transparent bg-red-100 text-red-700 dark:bg-red-300/15 dark:text-red-300",
    disabled:
      "border-transparent bg-slate-100 text-slate-500 dark:bg-slate-300/15 dark:text-slate-400",
  };

  const badgeLabels: Record<string, string> = {
    registered: "Registered",
    not_registered: "Not Registered",
    failed: "Failed",
    disabled: "Disabled",
  };

  let badgeLabel = $derived(
    status ? badgeLabels[status.status] ?? status.status : "",
  );
  let badgeColor = $derived(
    status ? badgeColors[status.status] ?? badgeColors.not_registered : "",
  );

  let isRegistered = $derived(status?.status === "registered");
  let isFailed = $derived(status?.status === "failed");
  let showRegister = $derived(!isRegistered && !isFailed);
  let showRepair = $derived(isRegistered || isFailed);
  let showUnregister = $derived(isRegistered || isFailed);

  function loadStatus() {
    if (!slug) return;
    loading = true;
    loadError = "";
    getProjectWebhookStatus(slug)
      .then((res) => {
        status = res;
        errorCode = null;
      })
      .catch((e: Error) => {
        status = null;
        loadError = e?.message || "Failed to load webhook status";
        errorCode = (e as { code?: number })?.code ?? null;
      })
      .finally(() => {
        loading = false;
      });
  }

  $effect(loadStatus);

  async function handleRegister() {
    if (busy) return;
    busy = true;
    try {
      const res = await registerProjectWebhook(slug);
      status = res;
      toast.success("Webhook registered");
    } catch (e: any) {
      toast.error(e?.message || "Failed to register webhook");
    } finally {
      busy = false;
      loadStatus();
    }
  }

  async function handleUnregister() {
    if (busy) return;
    busy = true;
    try {
      const res = await unregisterProjectWebhook(slug);
      status = res;
      showUnregisterDialog = false;
      toast.success("Webhook unregistered");
    } catch (e: any) {
      toast.error(e?.message || "Failed to unregister webhook");
    } finally {
      busy = false;
      loadStatus();
    }
  }
</script>

<Card.Root>
  <Card.CardHeader>
    <div class="flex items-center justify-between gap-2">
      <Card.CardTitle>Webhook</Card.CardTitle>
      {#if status}
        <Badge.Root class={badgeColor}>{badgeLabel}</Badge.Root>
      {/if}
    </div>
  </Card.CardHeader>
  <Card.CardContent>
    {#if loading}
      <p class="text-muted-foreground text-sm">Loading...</p>
    {:else if loadError}
      {#if oauthRevoked}
        <div
          class="border-destructive/40 bg-destructive/5 flex flex-col gap-2 rounded-md border p-3"
        >
          <p class="text-destructive text-sm font-medium">
            Bitbucket OAuth connection revoked
          </p>
          <p class="text-muted-foreground text-xs">{loadError}</p>
          <div class="flex gap-2">
            <Button
              size="sm"
              onclick={() => navigate("/admin")}
            >
              Reconnect OAuth
            </Button>
            <Button size="sm" variant="outline" onclick={loadStatus}>
              Retry
            </Button>
          </div>
        </div>
      {:else if permissionDenied}
        <p class="text-destructive text-sm">{loadError}</p>
        <p class="text-muted-foreground text-xs">
          The Bitbucket app needs repository admin permissions to manage
          webhooks.
        </p>
      {:else}
        <p class="text-destructive text-sm">{loadError}</p>
      {/if}
    {:else if status}
      <div class="flex flex-col gap-2">
        {#if status.callback_url}
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-xs font-medium">
              Callback URL
            </span>
            <code
              class="border-input bg-muted text-muted-foreground block overflow-x-auto rounded border px-2 py-1 font-mono text-xs"
            >
              {status.callback_url}
            </code>
          </div>
        {/if}
        {#if status.failure_reason}
          <p class="text-destructive text-sm">{status.failure_reason}</p>
        {/if}
      </div>
    {/if}
  </Card.CardContent>
  {#if !loading && (status || loadError)}
    <Card.CardFooter class="justify-end gap-2">
      {#if loadError && !status && !oauthRevoked}
        <Button size="sm" variant="outline" onclick={loadStatus}>
          Retry
        </Button>
      {/if}
      {#if status}
        {#if showRegister}
          <Button size="sm" disabled={busy} onclick={handleRegister}>
            {busy ? "..." : "Register"}
          </Button>
        {/if}
        {#if showRepair}
          <Button size="sm" variant="outline" disabled={busy} onclick={handleRegister}>
            {busy ? "..." : "Repair"}
          </Button>
        {/if}
        {#if showUnregister}
          <Button
            size="sm"
            variant="destructive"
            disabled={busy}
            onclick={() => (showUnregisterDialog = true)}
          >
            Unregister
          </Button>
        {/if}
      {/if}
    </Card.CardFooter>
  {/if}
</Card.Root>

<Dialog.Root
  bind:open={showUnregisterDialog}
  title="Unregister webhook?"
  description={`Remove the Bitbucket push webhook for "${slug}"?`}
>
  <p class="text-muted-foreground text-sm">
    Push events will no longer be delivered to this instance until the webhook
    is registered again.
  </p>
  {#snippet footer()}
    <Button
      variant="ghost"
      onclick={() => (showUnregisterDialog = false)}
    >
      Cancel
    </Button>
    <Button variant="destructive" onclick={handleUnregister} disabled={busy}>
      {busy ? "Unregistering..." : "Unregister"}
    </Button>
  {/snippet}
</Dialog.Root>
