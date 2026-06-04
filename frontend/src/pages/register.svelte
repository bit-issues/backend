<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import * as Card from "$lib/components/ui/card";
  import { register, isAuthenticated } from "$lib/stores/auth.svelte";
  import { navigate } from "$lib/router/routes";
  import { ApiErrorResponse } from "$lib/api/client";

  let { params = {} }: { params?: Record<string, string> } = $props();

  let email = $state("");
  let password = $state("");
  let confirmPassword = $state("");
  let loading = $state(false);
  let error = $state("");
  let success = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    loading = true;
    error = "";

    if (password !== confirmPassword) {
      error = "Passwords do not match";
      loading = false;
      return;
    }

    try {
      await register(email, password);
      success = true;
    } catch (err) {
      if (err instanceof ApiErrorResponse) {
        if (err.errors?.length) {
          error = err.errors.map((e) => `${e.field}: ${e.message}`).join(", ");
        } else {
          error = err.message || "Registration failed";
        }
      } else {
        error = "An unexpected error occurred";
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (isAuthenticated()) {
      navigate("/");
    }
  });
</script>

<div class="flex min-h-screen items-center justify-center p-4">
  {#if success}
    <Card.Root size="sm" class="w-full max-w-sm">
      <Card.CardHeader>
        <Card.CardTitle class="text-xl">Account created</Card.CardTitle>
        <Card.CardDescription>
          Your account has been created and is pending activation. Please
          contact your administrator to activate your account.
        </Card.CardDescription>
      </Card.CardHeader>
      <Card.CardContent>
        <Button onclick={() => navigate("/login")} class="w-full">
          Back to Sign in
        </Button>
      </Card.CardContent>
    </Card.Root>
  {:else}
    <Card.Root size="sm" class="w-full max-w-sm">
      <Card.CardHeader>
        <Card.CardTitle class="text-xl">Create account</Card.CardTitle>
        <Card.CardDescription
          >Register for a new BitIssues account</Card.CardDescription
        >
      </Card.CardHeader>
      <Card.CardContent>
        <form onsubmit={handleSubmit} class="flex flex-col gap-4">
          <div class="flex flex-col gap-2">
            <label for="email" class="text-sm font-medium">Email</label>
            <Input
              id="email"
              type="email"
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
              placeholder="At least 8 characters"
              bind:value={password}
              required
              minlength={8}
              disabled={loading}
            />
          </div>
          <div class="flex flex-col gap-2">
            <label for="confirmPassword" class="text-sm font-medium"
              >Confirm password</label
            >
            <Input
              id="confirmPassword"
              type="password"
              placeholder="Repeat your password"
              bind:value={confirmPassword}
              required
              minlength={8}
              disabled={loading}
            />
          </div>
          {#if error}
            <p class="text-destructive text-sm">{error}</p>
          {/if}
          <Button type="submit" disabled={loading} class="w-full">
            {loading ? "Creating account..." : "Create account"}
          </Button>
        </form>
      </Card.CardContent>
      <Card.CardFooter class="justify-center">
        <p class="text-muted-foreground text-sm">
          Already have an account?
          <button
            type="button"
            onclick={() => navigate("/login")}
            class="text-primary cursor-pointer underline-offset-4 hover:underline"
          >
            Sign in
          </button>
        </p>
      </Card.CardFooter>
    </Card.Root>
  {/if}
</div>
