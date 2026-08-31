<script lang="ts">
  import { onMount } from "svelte";
  import BitbucketOAuthCard from "$lib/components/BitbucketOAuthCard.svelte";
  import { toast } from "$lib/toast";

  const OAUTH_ERROR_MESSAGES: Record<string, string> = {
    access_denied: "Bitbucket authorization was denied",
    missing_params: "The Bitbucket callback was missing required parameters",
    exchange_failed: "Bitbucket rejected the authorization code",
    invalid_state: "The connection session expired or was already used. Try again.",
  };

  onMount(() => {
    // The Bitbucket OAuth callback redirects to /#/admin?oauth=success|error.
    // With the hash router the result lives in the fragment, so read it from
    // window.location.hash (falling back to search for robustness).
    const hashQuery = window.location.hash.split("?")[1] ?? "";
    const params = new URLSearchParams(hashQuery || window.location.search);
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

    // Consume the one-shot oauth/reason params so a page refresh cannot replay
    // the toast. Strip them from BOTH the hash fragment and window.location.search.
    const hashParts = window.location.hash.slice(1).split("?");
    const route = hashParts[0] || "/admin";
    const hashParams = new URLSearchParams(hashParts[1] ?? "");
    hashParams.delete("oauth");
    hashParams.delete("reason");

    const searchParams = new URLSearchParams(window.location.search);
    searchParams.delete("oauth");
    searchParams.delete("reason");

    const searchStr = searchParams.toString();
    const hashQueryStr = hashParams.toString();
    const newURL =
      window.location.pathname +
      (searchStr ? "?" + searchStr : "") +
      "#" + route +
      (hashQueryStr ? "?" + hashQueryStr : "");

    window.history.replaceState(null, "", newURL);
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
