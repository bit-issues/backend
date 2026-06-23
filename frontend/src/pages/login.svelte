<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import * as Card from "$lib/components/ui/card";
  import { login, isAuthenticated, conditionalPasskeyLogin, loginWithPasskey } from "$lib/stores/auth.svelte";
  import { navigate } from "$lib/router/routes";
  import { toast } from "$lib/toast";
  import { ApiErrorResponse } from "$lib/api/client";

  let { params = {} }: { params?: Record<string, string> } = $props();

  let email = $state("");
  let password = $state("");
  let loading = $state(false);
  let passkeyLoading = $state(false);
  let error = $state("");
  let passkeySupported = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    loading = true;
    error = "";

    try {
      await login(email, password);
      toast.success("Logged in successfully");
      navigate("/");
    } catch (err) {
      if (err instanceof ApiErrorResponse) {
        if (err.code === 403) {
          navigate("/pending");
          return;
        }
        error = err.message || "Login failed";
      } else {
        error = "An unexpected error occurred";
      }
    } finally {
      loading = false;
    }
  }

  async function handlePasskeyLogin() {
    passkeyLoading = true;
    error = "";

    try {
      await loginWithPasskey();
      toast.success("Logged in successfully");
      navigate("/");
    } catch (err) {
      if (err instanceof Error) {
        error = err.message || "Passkey login failed";
      } else {
        error = "Passkey login failed";
      }
    } finally {
      passkeyLoading = false;
    }
  }

  $effect(() => {
    if (isAuthenticated()) {
      navigate("/");
    }
  });

  $effect(() => {
    passkeySupported = typeof navigator !== 'undefined' && 'credentials' in navigator && 'PublicKeyCredential' in window;
    if (passkeySupported) {
      conditionalPasskeyLogin().then(() => {
        if (isAuthenticated()) {
          toast.success("Logged in successfully");
          navigate("/");
        }
      });
    }
  });
</script>

<div class="flex min-h-screen items-center justify-center p-4">
  <Card.Root size="sm" class="w-full max-w-sm">
    <Card.CardHeader>
      <Card.CardTitle class="text-xl">Sign in</Card.CardTitle>
      <Card.CardDescription
        >Enter your credentials to access BitIssues</Card.CardDescription
      >
    </Card.CardHeader>
    <Card.CardContent>
      <form onsubmit={handleSubmit} class="flex flex-col gap-4">
        <div class="flex flex-col gap-2">
          <label for="email" class="text-sm font-medium">Email</label>
          <Input
            id="email"
            type="email"
            autocomplete="username webauthn"
            placeholder="email@example.com"
            bind:value={email}
            required
            disabled={loading}
          />
        </div>
        <div class="flex flex-col gap-2">
          <label for="password" class="text-sm font-medium">Password</label>
          <Input
            id="password"
            type="password"
            autocomplete="current-password"
            placeholder="Enter your password"
            bind:value={password}
            required
            disabled={loading}
          />
        </div>
        {#if error}
          <p class="text-destructive text-sm">{error}</p>
        {/if}
        <Button type="submit" disabled={loading} class="w-full">
          {loading ? "Signing in..." : "Sign in"}
        </Button>
      </form>

      {#if passkeySupported}
        <div class="relative my-4">
          <div class="absolute inset-0 flex items-center">
            <span class="w-full border-t"></span>
          </div>
          <div class="relative flex justify-center text-xs uppercase">
            <span class="bg-card px-2 text-muted-foreground">or</span>
          </div>
        </div>

        <Button
          variant="outline"
          onclick={handlePasskeyLogin}
          disabled={passkeyLoading}
          class="w-full"
        >
          {passkeyLoading ? "Verifying..." : "Sign in with passkey"}
        </Button>
      {/if}
    </Card.CardContent>
    <Card.CardFooter class="justify-center">
      <p class="text-muted-foreground text-sm">
        Don't have an account?
        <button
          type="button"
          onclick={() => navigate("/register")}
          class="text-primary cursor-pointer underline-offset-4 hover:underline"
        >
          Register
        </button>
      </p>
    </Card.CardFooter>
  </Card.Root>
</div>
