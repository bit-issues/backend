<script lang="ts">
  import { searchUsers } from "$lib/api/users";
  import type { UserBrief } from "$lib/types/api";

  let {
    value = $bindable<number | null>(null),
    placeholder = "Search user...",
    initialName = "",
    id = "",
  }: {
    value?: number | null;
    placeholder?: string;
    initialName?: string;
    id?: string;
  } = $props();

  let query = $state("");
  let results = $state<UserBrief[]>([]);
  let open = $state(false);
  let loading = $state(false);
  let selectedName = $state("");
  let timer: ReturnType<typeof setTimeout> | null = null;
  let requestSeq = 0;

  $effect(() => {
    if (initialName) selectedName = initialName;
  });

  function doSearch(q: string) {
    const seq = ++requestSeq;
    if (!q.trim()) {
      results = [];
      open = false;
      return;
    }
    loading = true;
    searchUsers(q, 10)
      .then((r) => {
        if (seq !== requestSeq) return;
        results = r.items;
        open = true;
      })
      .catch(() => {
        if (seq !== requestSeq) return;
        results = [];
      })
      .finally(() => {
        if (seq !== requestSeq) return;
        loading = false;
      });
  }

  function handleInput(e: Event) {
    const q = (e.target as HTMLInputElement).value;
    query = q;
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => doSearch(q), 300);
  }

  function select(user: UserBrief) {
    value = user.id;
    selectedName = user.name;
    query = "";
    results = [];
    open = false;
  }

  function clear() {
    value = null;
    selectedName = "";
    query = "";
    results = [];
    open = false;
  }

  function handleBlur() {
    setTimeout(() => {
      open = false;
    }, 200);
  }
</script>

<div class="relative">
  {#if selectedName}
    <div
      class="border-input flex h-9 items-center gap-1.5 rounded-md border px-2.5 py-1 text-sm"
    >
      <span class="flex-1">{selectedName}</span>
      <button
        type="button"
        onclick={clear}
        class="text-muted-foreground hover:text-foreground ml-1 cursor-pointer leading-none text-lg"
      >
        &times;
      </button>
    </div>
  {:else}
    <input
      type="text"
      {id}
      class="border-input bg-background ring-offset-background focus-visible:ring-ring h-9 w-full rounded-md border px-2.5 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
      {placeholder}
      value={query}
      oninput={handleInput}
      onfocus={() => {
        if (results.length) open = true;
      }}
      onblur={handleBlur}
    />
  {/if}

  {#if open && results.length > 0}
    <div
      class="bg-popover text-popover-foreground absolute z-50 mt-1 w-full rounded-md border shadow-md"
    >
      <ul class="max-h-48 overflow-auto py-1 bg-popover">
        {#each results as user}
          <li>
            <button
              type="button"
              onclick={() => select(user)}
              class="bg-popover hover:bg-muted w-full px-2.5 py-1.5 text-left text-sm cursor-pointer"
            >
              {user.name}
              <span class="text-muted-foreground ml-2 text-xs">{user.role}</span
              >
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  {#if loading}
    <div class="text-muted-foreground absolute right-2.5 top-1.5 text-xs">
      Searching...
    </div>
  {/if}
</div>
