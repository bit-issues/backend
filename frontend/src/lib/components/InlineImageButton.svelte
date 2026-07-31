<script lang="ts">
  let {
    uploading = false,
    disabled = false,
    onclick,
  }: {
    uploading?: boolean;
    disabled?: boolean;
    onclick: () => void;
  } = $props();
</script>

<button
  type="button"
  {onclick}
  disabled={disabled || uploading}
  class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-50"
  title="Insert image"
>
  {#if uploading}
    <span
      class="size-3 animate-spin rounded-full border-2 border-current border-t-transparent"
    ></span>
    Uploading...
  {:else}
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      class="size-4"
      ><path
        fill-rule="evenodd"
        d="M1 5.25A2.25 2.25 0 0 1 3.25 3h13.5A2.25 2.25 0 0 1 19 5.25v9.5A2.25 2.25 0 0 1 16.75 17H3.25A2.25 2.25 0 0 1 1 14.75v-9.5Zm1.5 5.81v3.69c0 .414.336.75.75.75h13.5a.75.75 0 0 0 .75-.75v-2.69l-2.22-2.219a.75.75 0 0 0-1.06 0l-1.91 1.909.47.47a.75.75 0 1 1-1.06 1.06L6.53 8.091a.75.75 0 0 0-1.06 0l-2.97 2.97ZM12 7a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z"
        clip-rule="evenodd"
      /></svg
    >
    Image
  {/if}
</button>
