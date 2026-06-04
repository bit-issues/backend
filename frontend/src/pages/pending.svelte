<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { navigate } from "$lib/router/routes";
  import ClockIcon from "@lucide/svelte/icons/clock";

  let { params = {} }: { params?: Record<string, string> } = $props();
</script>

<div class="flex min-h-screen items-center justify-center p-4">
  <Card.Root size="sm" class="w-full max-w-sm">
    <Card.CardHeader>
      <div class="mb-2 flex justify-center">
        <ClockIcon class="text-muted-foreground size-10" />
      </div>
      <Card.CardTitle class="text-center text-xl"
        >Pending activation</Card.CardTitle
      >
      <Card.CardDescription class="text-center">
        Your account is pending activation. Please contact your administrator to
        activate your account before you can sign in.
      </Card.CardDescription>
    </Card.CardHeader>
    <Card.CardContent>
      <Button onclick={() => navigate("/login")} class="w-full">
        Back to Sign in
      </Button>
    </Card.CardContent>
  </Card.Root>
</div>
