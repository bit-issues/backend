<script lang="ts">
  import { onMount } from "svelte";
  import { getUser } from "$lib/stores/auth.svelte";
  import { listUsers, updateUser } from "$lib/api/users";
  import { toast } from "$lib/toast";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import * as Table from "$lib/components/ui/table";
  import * as Badge from "$lib/components/ui/badge";
  import * as Dialog from "$lib/components/ui/dialog";
  import type { User } from "$lib/types/api";

  const PAGE_SIZE = 20;

  let users = $state<User[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let error = $state("");

  let filterStatus = $state("");
  let filterRole = $state("");
  let page = $state(1);

  let showEditDialog = $state(false);
  let editingUser = $state<User | null>(null);
  let editRole = $state<"admin" | "user">("user");
  let editStatus = $state<"pending" | "active" | "blocked">("pending");
  let editSaving = $state(false);

  let activatingId = $state<number | null>(null);

  let totalPages = $derived(Math.max(1, Math.ceil(total / PAGE_SIZE)));

  const statusBadgeColors: Record<string, string> = {
    active:
      "border-transparent bg-green-100 text-green-700 dark:bg-green-300/15 dark:text-green-300",
    pending:
      "border-transparent bg-amber-100 text-amber-700 dark:bg-amber-300/15 dark:text-amber-300",
    blocked:
      "border-transparent bg-red-100 text-red-700 dark:bg-red-300/15 dark:text-red-300",
  };

  const roleBadgeColors: Record<string, string> = {
    admin:
      "border-transparent bg-purple-100 text-purple-700 dark:bg-purple-300/15 dark:text-purple-300",
    user: "border-transparent bg-blue-100 text-blue-700 dark:bg-blue-300/15 dark:text-blue-300",
  };

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString();
  }

  async function loadUsers() {
    loading = true;
    error = "";
    try {
      const res = await listUsers({
        status: filterStatus || undefined,
        role: filterRole || undefined,
        limit: PAGE_SIZE,
        offset: (page - 1) * PAGE_SIZE,
      });
      users = res.items;
      total = res.total;
    } catch (e: any) {
      error = e?.message || "Failed to load users";
      users = [];
    } finally {
      loading = false;
    }
  }

  function onFilterChange() {
    page = 1;
    loadUsers();
  }

  function resetFilters() {
    filterStatus = "";
    filterRole = "";
    page = 1;
    loadUsers();
  }

  function openEdit(user: User) {
    editingUser = user;
    editRole = user.role;
    editStatus = user.status;
    showEditDialog = true;
  }

  function closeEdit() {
    showEditDialog = false;
    editingUser = null;
  }

  async function saveEdit() {
    if (!editingUser || editSaving) return;
    editSaving = true;
    try {
      const updated = await updateUser(editingUser.id, {
        role: editRole,
        status: editStatus,
      });
      const idx = users.findIndex((u) => u.id === editingUser!.id);
      if (idx !== -1) users[idx] = updated;
      toast.success("User updated");

      if (getUser()?.id === editingUser.id) {
        const stored = localStorage.getItem("user");
        if (stored) {
          const u = JSON.parse(stored);
          u.role = updated.role;
          u.status = updated.status;
          localStorage.setItem("user", JSON.stringify(u));
        }
      }

      closeEdit();
    } catch (e: any) {
      toast.error(e?.message || "Failed to update user");
    } finally {
      editSaving = false;
    }
  }

  async function handleActivate(user: User) {
    if (activatingId) return;
    activatingId = user.id;
    try {
      const updated = await updateUser(user.id, { status: "active" });
      const idx = users.findIndex((u) => u.id === user.id);
      if (idx !== -1) users[idx] = updated;
      toast.success(`${user.name} activated`);
    } catch (e: any) {
      toast.error(e?.message || "Failed to activate user");
    } finally {
      activatingId = null;
    }
  }

  onMount(loadUsers);
</script>

<div class="flex flex-col gap-4 p-6">
  <div>
    <h1 class="text-2xl font-bold">Users</h1>
    <p class="text-muted-foreground text-sm">Manage user accounts</p>
  </div>

  <Card.Root>
    <Card.CardContent class="pt-6">
      <div class="flex flex-wrap items-end gap-3">
        <div class="flex flex-col gap-1.5">
          <span class="text-muted-foreground text-xs font-medium">Status</span>
          <select
            class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
            bind:value={filterStatus}
            onchange={onFilterChange}
          >
            <option value="">All</option>
            <option value="pending">Pending</option>
            <option value="active">Active</option>
            <option value="blocked">Blocked</option>
          </select>
        </div>
        <div class="flex flex-col gap-1.5">
          <span class="text-muted-foreground text-xs font-medium">Role</span>
          <select
            class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
            bind:value={filterRole}
            onchange={onFilterChange}
          >
            <option value="">All</option>
            <option value="admin">Admin</option>
            <option value="user">User</option>
          </select>
        </div>
        <Button variant="ghost" size="sm" onclick={resetFilters}>Reset</Button>
      </div>
    </Card.CardContent>
  </Card.Root>

  <Card.Root>
    <Table.Root>
      <Table.Header>
        <Table.Row>
          <Table.Head class="w-12">#</Table.Head>
          <Table.Head>Name</Table.Head>
          <Table.Head>Email</Table.Head>
          <Table.Head>Role</Table.Head>
          <Table.Head>Status</Table.Head>
          <Table.Head>Created</Table.Head>
          <Table.Head class="w-24">Actions</Table.Head>
        </Table.Row>
      </Table.Header>
      <Table.Body>
        {#if loading}
          <Table.Row>
            <td
              colspan={7}
              class="text-muted-foreground p-2 py-8 text-center align-middle"
            >
              Loading...
            </td>
          </Table.Row>
        {:else if error}
          <Table.Row>
            <td
              colspan={7}
              class="text-destructive p-2 py-8 text-center align-middle"
            >
              {error}
            </td>
          </Table.Row>
        {:else if users.length === 0}
          <Table.Row>
            <td
              colspan={7}
              class="text-muted-foreground p-2 py-8 text-center align-middle italic"
            >
              No users found
            </td>
          </Table.Row>
        {:else}
          {#each users as user}
            <Table.Row>
              <Table.Cell class="text-muted-foreground text-xs">
                {user.id}
              </Table.Cell>
              <Table.Cell class="font-medium">{user.name}</Table.Cell>
              <Table.Cell class="text-muted-foreground">{user.email}</Table.Cell
              >
              <Table.Cell>
                <Badge.Root class={roleBadgeColors[user.role]}>
                  {user.role}
                </Badge.Root>
              </Table.Cell>
              <Table.Cell>
                <Badge.Root class={statusBadgeColors[user.status]}>
                  {user.status}
                </Badge.Root>
              </Table.Cell>
              <Table.Cell class="text-muted-foreground text-xs">
                {formatDate(user.created_at)}
              </Table.Cell>
              <Table.Cell>
                <div class="flex gap-1">
                  {#if user.status === "pending"}
                    <Button
                      variant="default"
                      size="sm"
                      disabled={activatingId === user.id}
                      onclick={() => handleActivate(user)}
                    >
                      {activatingId === user.id ? "..." : "Activate"}
                    </Button>
                  {/if}
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() => openEdit(user)}
                  >
                    Edit
                  </Button>
                </div>
              </Table.Cell>
            </Table.Row>
          {/each}
        {/if}
      </Table.Body>
    </Table.Root>

    {#if totalPages > 1}
      <div
        class="flex items-center justify-center gap-2 border-t border-border px-4 py-3"
      >
        <Button
          variant="outline"
          size="sm"
          disabled={page <= 1}
          onclick={() => {
            page--;
            loadUsers();
          }}
        >
          Previous
        </Button>
        <span class="text-muted-foreground text-sm">
          Page {page} of {totalPages}
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={page >= totalPages}
          onclick={() => {
            page++;
            loadUsers();
          }}
        >
          Next
        </Button>
      </div>
    {/if}
  </Card.Root>
</div>

<Dialog.Root
  bind:open={showEditDialog}
  title="Edit User"
  description={editingUser?.name ?? ""}
>
  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-2">
      <label for="edit-role" class="text-sm font-medium">Role</label>
      <select
        id="edit-role"
        class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
        bind:value={editRole}
      >
        <option value="admin">Admin</option>
        <option value="user">User</option>
      </select>
    </div>
    <div class="flex flex-col gap-2">
      <label for="edit-status" class="text-sm font-medium">Status</label>
      <select
        id="edit-status"
        class="border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:outline-none"
        bind:value={editStatus}
      >
        <option value="pending">Pending</option>
        <option value="active">Active</option>
        <option value="blocked">Blocked</option>
      </select>
    </div>
  </div>
  {#snippet footer()}
    <Button variant="ghost" onclick={closeEdit}>Cancel</Button>
    <Button onclick={saveEdit} disabled={editSaving}>
      {editSaving ? "Saving..." : "Save"}
    </Button>
  {/snippet}
</Dialog.Root>
