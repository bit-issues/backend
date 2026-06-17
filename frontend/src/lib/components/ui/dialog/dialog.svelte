<script lang="ts">
  import { fade } from "svelte/transition";
  import { tick } from "svelte";

  let {
    open = $bindable(false),
    title = "",
    description = "",
    children,
    footer,
  }: {
    open?: boolean;
    title?: string;
    description?: string;
    children?: import("svelte").Snippet;
    footer?: import("svelte").Snippet;
  } = $props();
  let dialogEl = $state<HTMLDivElement | null>(null);
  let previouslyFocused = $state<HTMLElement | null>(null);

  $effect(() => {
    if (open) {
      previouslyFocused = document.activeElement as HTMLElement | null;
      tick().then(() => dialogEl?.focus());
    } else {
      previouslyFocused?.focus();
    }
  });

  function close() {
    open = false;
  }

  function onContentClick(e: MouseEvent) {
    e.stopPropagation();
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <!-- svelte-ignore a11y_interactive_supports_focus -->
  <div
    bind:this={dialogEl}
    class="fixed inset-0 z-50 flex items-center justify-center"
    transition:fade={{ duration: 150 }}
    onclick={close}
    onkeydown={(e) => {
      if (e.key === "Escape") close();
    }}
    role="dialog"
    tabindex="-1"
    aria-modal="true"
    aria-label={title || "Dialog"}
  >
    <div class="fixed inset-0 bg-black/50"></div>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="bg-background relative z-50 mx-auto w-full max-w-lg rounded-lg border p-6 shadow-lg"
      role="presentation"
      onclick={onContentClick}
    >
      <button
        type="button"
        onclick={close}
        class="text-muted-foreground hover:text-foreground absolute top-4 right-4 inline-flex items-center justify-center rounded-full p-1 transition-colors"
        aria-label="Close"
      >
        ✕
      </button>
      {#if title}
        <h2 class="mb-1 text-lg font-semibold">{title}</h2>
      {/if}
      {#if description}
        <p class="text-muted-foreground mb-4 text-sm">{description}</p>
      {/if}
      <div>
        {@render children?.()}
      </div>
      {#if footer}
        <div class="mt-6 flex items-center justify-end gap-2">
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}
