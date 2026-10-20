<script lang="ts">
  import { ApiErrorResponse } from "$lib/api/client";
  import { getProject } from "$lib/api/projects";
  import {
    getProjectWebhook,
    registerProjectWebhook,
    unregisterProjectWebhook,
  } from "$lib/api/webhooks";
  import { navigate } from "$lib/router/routes";
  import { toast } from "$lib/toast";
  import { Button } from "$lib/components/ui/button";
  import * as Badge from "$lib/components/ui/badge";
  import * as Card from "$lib/components/ui/card";
  import * as Dialog from "$lib/components/ui/dialog";
  import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
  import type {
    Project,
    WebhookStatus,
    WebhookStatusValue,
  } from "$lib/types/api";

  let { params = {} }: { params?: Record<string, string> } = $props();
  let slug = $derived(params.slug || "");

  let project = $state<Project | null>(null);
  let status = $state<WebhookStatus | null>(null);
  let loading = $state(true);
  let loadError = $state("");
  let busyAction = $state<"register" | "unregister" | null>(null);
  let showReconnect = $state(false);
  let showUnregisterDialog = $state(false);

  const BADGE: Record<
    WebhookStatusValue,
    { label: string; class: string }
  > = {
    registered: {
      label: "Registered",
      class:
        "border-transparent bg-green-100 text-green-700 dark:bg-green-300/15 dark:text-green-300",
    },
    not_registered: {
      label: "Not registered",
      class:
        "border-transparent bg-gray-100 text-gray-600 dark:bg-gray-300/15 dark:text-gray-300",
    },
    failed: {
      label: "Failed",
      class:
        "border-transparent bg-red-100 text-red-700 dark:bg-red-300/15 dark:text-red-300",
    },
    unavailable: {
      label: "Unavailable",
      class:
        "border-transparent bg-amber-100 text-amber-700 dark:bg-amber-300/15 dark:text-amber-300",
    },
  };

  const PRIMARY_LABEL: Record<WebhookStatusValue, string> = {
    registered: "Re-register",
    not_registered: "Register webhook",
    failed: "Repair webhook",
    unavailable: "Register webhook",
  };

  function formatDate(iso?: string | null): string {
    if (!iso) return "-";
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
  }

  function loadStatus() {
    if (!slug) return;
    loading = true;
    loadError = "";
    getProjectWebhook(slug)
      .then((res) => {
        status = res;
        showReconnect = false;
      })
      .catch((e: unknown) => {
        status = null;
        if (e instanceof ApiErrorResponse && e.code === 409) {
          showReconnect = true;
        }
        loadError =
          e instanceof Error && e.message
            ? e.message
            : "Failed to load webhook status";
      })
      .finally(() => {
        loading = false;
      });
  }

  function loadProject() {
    if (!slug) return;
    project = null;
    getProject(slug)
      .then((p) => {
        project = p;
      })
      .catch(() => {
        project = null;
      });
  }

  $effect(loadStatus);
  $effect(loadProject);

  let badge = $derived(status ? BADGE[status.status] : null);
  let hasHook = $derived(
    status?.status === "registered" || status?.status === "failed",
  );
  let unavailable = $derived(status?.status === "unavailable");
  let busy = $derived(busyAction !== null);

  function handleActionError(e: unknown) {
    if (e instanceof ApiErrorResponse && e.code === 409) {
      showReconnect = true;
      toast.error(e.message);
      return;
    }
    toast.error(e instanceof Error && e.message ? e.message : "Request failed");
  }

  async function handleRegister() {
    if (busy || !slug || !status) return;
    busyAction = "register";
    try {
      await registerProjectWebhook(slug);
      toast.success("Webhook registered");
      loadStatus();
    } catch (e) {
      handleActionError(e);
    } finally {
      busyAction = null;
    }
  }

  async function handleUnregister() {
    if (busy || !slug) return;
    busyAction = "unregister";
    try {
      await unregisterProjectWebhook(slug);
      showUnregisterDialog = false;
      toast.success("Webhook removed");
      loadStatus();
    } catch (e) {
      handleActionError(e);
    } finally {
      busyAction = null;
    }
  }
</script>

<div class="mx-auto flex max-w-3xl flex-col gap-4 p-6">
  <div class="flex items-center gap-3">
    <Button
      variant="outline"
      size="sm"
      onclick={() => navigate("/admin/projects")}
    >
      <ArrowLeftIcon class="size-4" />
      Back
    </Button>
    <div>
      <h1 class="text-2xl font-bold">{project?.name ?? slug}</h1>
      <p class="text-muted-foreground text-sm">
        Bitbucket webhook for {slug}
      </p>
    </div>
  </div>

  <Card.Root>
    <Card.CardHeader>
      <div class="flex items-center justify-between gap-2">
        <Card.CardTitle>Webhook status</Card.CardTitle>
        {#if badge}
          <Badge.Root class={badge.class}>{badge.label}</Badge.Root>
        {/if}
      </div>
      <Card.CardDescription>
        Live status from Bitbucket. Nothing is stored locally; each read
        queries the Bitbucket API.
      </Card.CardDescription>
    </Card.CardHeader>
    <Card.CardContent>
      {#if showReconnect}
        <div
          class="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-md border border-amber-300/50 bg-amber-50 p-3 dark:bg-amber-300/10"
        >
          <p class="text-sm text-amber-800 dark:text-amber-200">
            Bitbucket OAuth is not connected. Reconnect to manage webhooks.
          </p>
          <Button size="sm" variant="outline" onclick={() => navigate("/admin")}>
            Reconnect
          </Button>
        </div>
      {/if}
      {#if loading && !status}
        <p class="text-muted-foreground text-sm">Loading...</p>
      {:else if loadError}
        <p class="text-destructive text-sm">{loadError}</p>
      {:else if status}
        <div class="flex flex-col gap-3">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-xs font-medium">
              Callback URL
            </span>
            <span class="font-mono text-sm break-all">
              {status.callback_url || "-"}
            </span>
          </div>
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-xs font-medium">
              Hook UUID
            </span>
            <span class="font-mono text-sm">
              {status.hook_uuid || "-"}
            </span>
          </div>
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-xs font-medium">
              Updated At
            </span>
            <span class="text-sm">{formatDate(status.updated_at)}</span>
          </div>
          {#if status.failure_reason}
            <div class="flex flex-col gap-1">
              <span class="text-muted-foreground text-xs font-medium">
                Problem
              </span>
              <span class="text-destructive text-sm">
                {status.failure_reason}
              </span>
            </div>
          {/if}
        </div>
      {/if}
    </Card.CardContent>
    <Card.CardFooter class="flex flex-wrap justify-between gap-2">
      <p class="text-muted-foreground text-xs">
        {#if unavailable}Resolve the problem above to manage the webhook.{/if}
      </p>
      <div class="flex gap-2">
        {#if loadError}
          <Button size="sm" variant="outline" onclick={loadStatus}>
            Retry
          </Button>
        {/if}
        {#if status}
          <Button
            size="sm"
            variant="outline"
            disabled={loading || busy}
            onclick={loadStatus}
          >
            {loading ? "Refreshing..." : "Refresh"}
          </Button>
          {#if hasHook}
            <Button
              size="sm"
              variant="destructive"
              disabled={busy}
              onclick={() => (showUnregisterDialog = true)}
            >
              {busyAction === "unregister" ? "Removing..." : "Unregister"}
            </Button>
          {/if}
          <Button
            size="sm"
            disabled={loading || busy || unavailable}
            onclick={handleRegister}
          >
            {busyAction === "register"
              ? "Registering..."
              : PRIMARY_LABEL[status.status]}
          </Button>
        {/if}
      </div>
    </Card.CardFooter>
  </Card.Root>
</div>

<Dialog.Root
  bind:open={showUnregisterDialog}
  title="Remove webhook?"
  description="Delete the Bitbucket push webhook for this project? Push events will stop updating tasks."
>
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (showUnregisterDialog = false)}>
      Cancel
    </Button>
    <Button variant="destructive" onclick={handleUnregister} disabled={busy}>
      {busyAction === "unregister" ? "Removing..." : "Remove"}
    </Button>
  {/snippet}
</Dialog.Root>
