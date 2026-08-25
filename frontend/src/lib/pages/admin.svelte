<script lang="ts">
  import { onMount } from "svelte";
  import BitbucketOAuthCard from "$lib/components/BitbucketOAuthCard.svelte";
  import { toast } from "$lib/toast";

  const OAUTH_ERROR_MESSAGES: Record<string, string> = {
    access_denied: "Bitbucket authorization was denied",
    missing_params: "The Bitbucket callback was missing required parameters",
    invalid_state: "The Bitbucket connection request was invalid or expired",
    exchange_failed: "Bitbucket rejected the authorization code",
    invalid_scope:
      "The Bitbucket app is missing the 'webhook' scope; check its configuration",
    save_failed: "Failed to store the Bitbucket connection",
  };

  onMount(() => {
    // The Bitbucket OAuth callback redirects to /admin?oauth=success|error.
    // The hash router ignores query strings, so they are read from the
    // location directly on mount.
    const params = new URLSearchParams(window.location.search);
    const outcome = params.get("oauth");
    if (outcome === "success") {
      toast.success("Connected to Bitbucket");
    } else if (outcome === "error") {
      const reason = params.get("reason") ?? "";
      toast.error(
        OAUTH_ERROR_MESSAGES[reason] ?? "Failed to connect to Bitbucket",
      );
    } else {
      return;
    }
    window.history.replaceState(null, "", window.location.pathname);
  });
</script>

<div class="flex flex-col gap-4 p-6">
  <div>
    <h1 class="text-2xl font-bold">Settings</h1>
    <p class="text-muted-foreground text-sm">
      Manage integrations and workspace settings
    </p>
  </div>
  <BitbucketOAuthCard />
</div>
