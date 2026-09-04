<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import * as Badge from "$lib/components/ui/badge";
  import * as Dialog from "$lib/components/ui/dialog";
  import {
    disconnectBitbucketOAuth,
    getBitbucketOAuthAuthorizeUrl,
    getBitbucketOAuthStatus,
  } from "$lib/api/oauth";
  import { toast } from "$lib/toast";
  import type { BitbucketOAuthStatus } from "$lib/types/api";

  let status = $state<BitbucketOAuthStatus | null>(null);
  let loading = $state(true);
  let loadError = $state("");
  let busy = $state(false);
  let showDisconnectDialog = $state(false);

  let connected = $derived(status?.connected === true);

  let badgeLabel = $derived(connected ? "Connected" : "Disconnected");
  let badgeColor = $derived(
    connected
      ? "border-transparent bg-green-100 text-green-700 dark:bg-green-300/15 dark:text-green-300"
      : "border-transparent bg-gray-100 text-gray-600 dark:bg-gray-300/15 dark:text-gray-300",
  );

  function formatDate(iso?: string): string {
    if (!iso) return "-";
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
  }

  function loadStatus() {
    loading = true;
    loadError = "";
    getBitbucketOAuthStatus()
      .then((res) => {
        status = res;
      })
      .catch((e: Error) => {
        status = null;
        loadError = e?.message || "Failed to load Bitbucket connection status";
      })
      .finally(() => {
        loading = false;
      });
  }

  $effect(loadStatus);

  async function handleConnect() {
    if (busy) return;
    busy = true;
    try {
      const { url } = await getBitbucketOAuthAuthorizeUrl();
      window.location.assign(url);
    } catch (e: any) {
      toast.error(e?.message || "Failed to start Bitbucket connection");
      busy = false;
    }
  }

  async function handleDisconnect() {
    if (busy) return;
    busy = true;
    try {
      await disconnectBitbucketOAuth();
      status = { connected: false };
      showDisconnectDialog = false;
      toast.success("Disconnected from Bitbucket");
    } catch (e: any) {
      toast.error(e?.message || "Failed to disconnect from Bitbucket");
    } finally {
      busy = false;
    }
  }
</script>

<Card.Root>
  <Card.CardHeader>
    <div class="flex items-center justify-between gap-2">
      <Card.CardTitle>Bitbucket OAuth</Card.CardTitle>
      {#if status}
        <Badge.Root class={badgeColor}>{badgeLabel}</Badge.Root>
      {/if}
    </div>
  </Card.CardHeader>
  <Card.CardContent>
    {#if loading}
      <p class="text-muted-foreground text-sm">Loading...</p>
    {:else if loadError}
      <p class="text-destructive text-sm">{loadError}</p>
    {:else if status}
      <div class="flex flex-col gap-2">
        <p class="text-muted-foreground text-sm">
          {#if connected}
            Webhook registration uses the connected Bitbucket app.
          {:else}
            Connect a Bitbucket app to manage repository webhooks.
          {/if}
        </p>
        {#if connected}
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-xs font-medium">
              Connected At
            </span>
            <span class="text-sm">{formatDate(status.connected_at)}</span>
          </div>
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-xs font-medium">
              Token Expires At
            </span>
            <span class="text-sm">{formatDate(status.expires_at)}</span>
          </div>
          {#if status.scopes?.length}
            <div class="flex flex-col gap-1">
              <span class="text-muted-foreground text-xs font-medium">
                Scopes
              </span>
              <span class="text-sm">{status.scopes.join(", ")}</span>
            </div>
          {/if}
        {/if}
      </div>
    {/if}
  </Card.CardContent>
  {#if !loading && (status || loadError)}
    <Card.CardFooter class="justify-end gap-2">
      {#if loadError && !status}
        <Button size="sm" variant="outline" onclick={loadStatus}>Retry</Button>
      {/if}
      {#if status}
        {#if connected}
          <Button
            size="sm"
            variant="destructive"
            disabled={busy}
            onclick={() => (showDisconnectDialog = true)}
          >
            Disconnect
          </Button>
        {:else}
          <Button size="sm" disabled={busy} onclick={handleConnect}>
            {busy ? "Connecting..." : "Connect with Bitbucket"}
          </Button>
        {/if}
      {/if}
    </Card.CardFooter>
  {/if}
</Card.Root>

<Dialog.Root
  bind:open={showDisconnectDialog}
  title="Disconnect from Bitbucket?"
  description="Remove the stored Bitbucket OAuth connection?"
>
  <p class="text-muted-foreground text-sm">
    Active repository webhooks will stop delivering push events after the
    Bitbucket token expires (about 2 hours). No remote webhooks are removed
    automatically.
  </p>
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (showDisconnectDialog = false)}>
      Cancel
    </Button>
    <Button variant="destructive" onclick={handleDisconnect} disabled={busy}>
      {busy ? "Disconnecting..." : "Disconnect"}
    </Button>
  {/snippet}
</Dialog.Root>
