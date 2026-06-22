<script lang="ts">
  import {
    listProjects,
    createProject,
    updateProject,
    deleteProject,
  } from "$lib/api/projects";
  import { toast } from "$lib/toast";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import * as Table from "$lib/components/ui/table";
  import * as Dialog from "$lib/components/ui/dialog";
  import { Input } from "$lib/components/ui/input";
  import SearchIcon from "@lucide/svelte/icons/search";
  import XIcon from "@lucide/svelte/icons/x";
  import type { Project } from "$lib/types/api";

  let projects = $state<Project[]>([]);
  let loading = $state(true);
  let error = $state("");
  let searchTerm = $state("");
  let debouncedSearch = $state("");
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;

  let showCreate = $state(false);
  let createName = $state("");
  let createRepoUrl = $state("");
  let createSaving = $state(false);

  let showEditDialog = $state(false);
  let editProject = $state<Project | null>(null);
  let editName = $state("");
  let editRepoUrl = $state("");
  let editSaving = $state(false);

  function handleSearchInput(value: string) {
    searchTerm = value;
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      debouncedSearch = value;
    }, 300);
  }

  function clearSearch() {
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = null;
    searchTerm = "";
    debouncedSearch = "";
  }

  $effect(() => {
    debouncedSearch;
    load();
  });

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString();
  }

  async function load() {
    loading = true;
    error = "";
    try {
      const res = await listProjects(100, 0, debouncedSearch || undefined);
      projects = res.items;
    } catch (e: any) {
      error = e?.message || "Failed to load projects";
    } finally {
      loading = false;
    }
  }

  function openCreate() {
    createName = "";
    createRepoUrl = "";
    showCreate = true;
  }

  async function handleCreate() {
    if (!createName.trim() || !createRepoUrl.trim() || createSaving) return;
    createSaving = true;
    try {
      const p = await createProject({
        name: createName.trim(),
        repo_url: createRepoUrl.trim(),
      });
      projects = [...projects, p];
      toast.success("Project created");
      showCreate = false;
    } catch (e: any) {
      toast.error(e?.message || "Failed to create project");
    } finally {
      createSaving = false;
    }
  }

  function openEdit(p: Project) {
    editProject = p;
    editName = p.name;
    editRepoUrl = p.repo_url;
    showEditDialog = true;
  }

  async function handleEdit() {
    if (!editProject || !editName.trim() || !editRepoUrl.trim() || editSaving)
      return;
    editSaving = true;
    try {
      const updated = await updateProject(editProject.id, {
        name: editName.trim(),
        repo_url: editRepoUrl.trim(),
      });
      const idx = projects.findIndex((p) => p.id === editProject!.id);
      if (idx !== -1) projects[idx] = updated;
      toast.success("Project updated");
      showEditDialog = false;
      editProject = null;
    } catch (e: any) {
      toast.error(e?.message || "Failed to update project");
    } finally {
      editSaving = false;
    }
  }

  async function handleDelete(p: Project) {
    if (!confirm(`Delete project "${p.name}"?`)) return;
    try {
      await deleteProject(p.id);
      projects = projects.filter((x) => x.id !== p.id);
      toast.success("Project deleted");
    } catch (e: any) {
      toast.error(e?.message || "Failed to delete project");
    }
  }
</script>

<div class="flex flex-col gap-4 p-6">
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-bold">Projects</h1>
      <p class="text-muted-foreground text-sm">Manage projects</p>
    </div>
    <Button onclick={openCreate}>Create Project</Button>
  </div>

  <div class="relative">
    <SearchIcon
      class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
    />
    <input
      type="text"
      placeholder="Search projects by name or slug..."
      value={searchTerm}
      oninput={(e) => handleSearchInput((e.target as HTMLInputElement).value)}
      class="w-full rounded-md border border-input bg-background py-2 pl-10 pr-10 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
    />
    {#if searchTerm}
      <button
        onclick={clearSearch}
        class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
        aria-label="Clear search"
      >
        <XIcon class="size-4" />
      </button>
    {/if}
  </div>

  <Dialog.Root bind:open={showCreate} title="Create Project">
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-2">
        <label for="create-name" class="text-sm font-medium">Name</label>
        <Input
          id="create-name"
          bind:value={createName}
          placeholder="Project name"
        />
      </div>
      <div class="flex flex-col gap-2">
        <label for="create-url" class="text-sm font-medium">Repo URL</label>
        <Input
          id="create-url"
          bind:value={createRepoUrl}
          placeholder="https://bitbucket.org/..."
        />
      </div>
    </div>
    {#snippet footer()}
      <Button variant="ghost" onclick={() => (showCreate = false)}>
        Cancel
      </Button>
      <Button
        onclick={handleCreate}
        disabled={!createName.trim() || !createRepoUrl.trim() || createSaving}
      >
        {createSaving ? "Creating..." : "Create"}
      </Button>
    {/snippet}
  </Dialog.Root>

  <Card.Root>
    <Table.Root>
      <Table.Header>
        <Table.Row>
          <Table.Head class="w-24">ID</Table.Head>
          <Table.Head>Name</Table.Head>
          <Table.Head>Repo URL</Table.Head>
          <Table.Head>Created</Table.Head>
          <Table.Head class="w-32">Actions</Table.Head>
        </Table.Row>
      </Table.Header>
      <Table.Body>
        {#if loading}
          <Table.Row>
            <td
              colspan={5}
              class="text-muted-foreground p-2 py-8 text-center align-middle"
            >
              Loading...
            </td>
          </Table.Row>
        {:else if error}
          <Table.Row>
            <td
              colspan={5}
              class="text-destructive p-2 py-8 text-center align-middle"
            >
              {error}
            </td>
          </Table.Row>
        {:else if projects.length === 0}
          <Table.Row>
            <td
              colspan={5}
              class="text-muted-foreground p-2 py-8 text-center align-middle italic"
            >
              {#if debouncedSearch}
                No projects found matching "{debouncedSearch}"
              {:else}
                No projects
              {/if}
            </td>
          </Table.Row>
        {:else}
          {#each projects as p}
            <Table.Row>
              <Table.Cell class="font-mono text-xs text-muted-foreground">
                {p.id}
              </Table.Cell>
              <Table.Cell class="font-medium">{p.name}</Table.Cell>
              <Table.Cell class="text-muted-foreground max-w-[300px] truncate">
                <a
                  href={p.repo_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="hover:underline"
                >
                  {p.repo_url}
                </a>
              </Table.Cell>
              <Table.Cell class="text-muted-foreground text-xs">
                {formatDate(p.created_at)}
              </Table.Cell>
              <Table.Cell>
                <div class="flex gap-1">
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() => openEdit(p)}
                  >
                    Edit
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onclick={() => handleDelete(p)}
                  >
                    Delete
                  </Button>
                </div>
              </Table.Cell>
            </Table.Row>
          {/each}
        {/if}
      </Table.Body>
    </Table.Root>
  </Card.Root>
</div>

<Dialog.Root
  bind:open={showEditDialog}
  title="Edit Project"
  description={editProject?.name ?? ""}
>
  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-2">
      <label for="edit-name" class="text-sm font-medium">Name</label>
      <Input id="edit-name" bind:value={editName} />
    </div>
    <div class="flex flex-col gap-2">
      <label for="edit-url" class="text-sm font-medium">Repo URL</label>
      <Input id="edit-url" bind:value={editRepoUrl} />
    </div>
  </div>
  {#snippet footer()}
    <Button variant="ghost" onclick={() => (showEditDialog = false)}>
      Cancel
    </Button>
    <Button
      onclick={handleEdit}
      disabled={!editName.trim() || !editRepoUrl.trim() || editSaving}
    >
      {editSaving ? "Saving..." : "Save"}
    </Button>
  {/snippet}
</Dialog.Root>
