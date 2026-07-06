<script lang="ts">
  import * as Table from "$lib/components/ui/table";
  import type { Task, Priority, Status, Kind } from "$lib/types/api";

  type SortField = "priority" | "status" | "due_date" | "created_at";

  let {
    tasks = [],
    sort,
    onSort,
    onTaskClick,
  }: {
    tasks: Task[];
    sort?: string;
    onSort?: (field: SortField) => void;
    onTaskClick?: (task: Task) => void;
  } = $props();

  const priorityStyles: Record<Priority, string> = {
    Blocker:
      "border-red-300 text-red-700 dark:border-red-600 dark:text-red-400",
    Critical:
      "border-orange-300 text-orange-700 dark:border-orange-600 dark:text-orange-400",
    Major:
      "border-amber-300 text-amber-700 dark:border-amber-600 dark:text-amber-400",
    Minor: "border-sky-300 text-sky-600 dark:border-sky-600 dark:text-sky-400",
    Trivial:
      "border-gray-200 text-gray-400 dark:border-gray-600 dark:text-gray-500",
  };

  const statusStyles: Record<Status, string> = {
    New: "bg-blue-700/8 text-blue-700 ring-1 ring-blue-200 dark:bg-blue-300/15 dark:text-blue-300 dark:ring-blue-700",
    Open: "bg-sky-700/8 text-sky-700 ring-1 ring-sky-200 dark:bg-sky-300/15 dark:text-sky-300 dark:ring-sky-700",
    "In Progress":
      "bg-amber-700/8 text-amber-700 ring-1 ring-amber-200 dark:bg-amber-300/15 dark:text-amber-300 dark:ring-amber-700",
    Resolved:
      "bg-emerald-700/8 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-300/15 dark:text-emerald-300 dark:ring-emerald-700",
    Closed:
      "bg-slate-500/8 text-slate-500 ring-1 ring-slate-200 dark:bg-slate-400/15 dark:text-slate-400 dark:ring-slate-600",
    Reopened:
      "bg-red-700/8 text-red-700 ring-1 ring-red-200 dark:bg-red-300/15 dark:text-red-300 dark:ring-red-700",
    Invalid:
      "bg-gray-400/8 text-gray-400 ring-1 ring-gray-200 dark:bg-gray-500/15 dark:text-gray-500 dark:ring-gray-600",
    Duplicate:
      "bg-gray-400/8 text-gray-400 ring-1 ring-gray-200 dark:bg-gray-500/15 dark:text-gray-500 dark:ring-gray-600",
    Wontfix:
      "bg-gray-400/8 text-gray-400 ring-1 ring-gray-200 dark:bg-gray-500/15 dark:text-gray-500 dark:ring-gray-600",
    "On Hold":
      "bg-orange-700/8 text-orange-700 ring-1 ring-orange-200 dark:bg-orange-300/15 dark:text-orange-300 dark:ring-orange-700",
  };

  const kindDot: Record<Kind, string> = {
    Bug: "bg-red-500",
    Enhancement: "bg-emerald-500",
    Task: "bg-blue-500",
    Proposal: "bg-purple-500",
  };

  const kindText: Record<Kind, string> = {
    Bug: "text-red-600 dark:text-red-400",
    Enhancement: "text-emerald-600 dark:text-emerald-400",
    Task: "text-blue-600 dark:text-blue-400",
    Proposal: "text-purple-600 dark:text-purple-400",
  };

  function isSorted(field: string): string {
    if (!sort) return "";
    if (sort === field) return "↑";
    if (sort === `-${field}`) return "↓";
    return "";
  }

  function toggleSort(field: SortField) {
    if (!onSort) return;
    if (sort === field) {
      onSort(`-${field}` as SortField);
    } else if (sort === `-${field}`) {
      onSort(field);
    } else {
      onSort(field);
    }
  }

  function formatDate(dateStr: string | null): string {
    if (!dateStr) return "—";
    const d = new Date(dateStr);
    return d.toLocaleDateString();
  }
</script>

<Table.Root>
  <Table.Header>
    <Table.Row>
      <Table.Head class="w-16">#</Table.Head>
      <Table.Head>Title</Table.Head>
      <Table.Head
        class="w-24 cursor-pointer"
        onclick={() => toggleSort("priority")}
      >
        Priority {isSorted("priority")}
      </Table.Head>
      <Table.Head
        class="w-28 cursor-pointer"
        onclick={() => toggleSort("status")}
      >
        Status {isSorted("status")}
      </Table.Head>
      <Table.Head class="w-24">Kind</Table.Head>
      <Table.Head class="w-32">Assignee</Table.Head>
      <Table.Head
        class="w-28 cursor-pointer"
        onclick={() => toggleSort("due_date")}
      >
        Due date {isSorted("due_date")}
      </Table.Head>
      <Table.Head
        class="w-32 cursor-pointer"
        onclick={() => toggleSort("created_at")}
      >
        Created {isSorted("created_at")}
      </Table.Head>
    </Table.Row>
  </Table.Header>
  <Table.Body>
    {#if tasks.length === 0}
      <Table.Row>
        <Table.Cell colspan={8} class="text-muted-foreground py-8 text-center">
          No tasks found
        </Table.Cell>
      </Table.Row>
    {:else}
      {#each tasks as task (task.id)}
        <Table.Row class="cursor-pointer" onclick={() => onTaskClick?.(task)}>
          <Table.Cell class="font-mono text-xs">
            {task.project_slug}-{task.number}
          </Table.Cell>
          <Table.Cell class="max-w-xs truncate font-medium">
            <a
              href="#/tasks/{task.id}"
              class="hover:underline"
              onclick={(e) => e.stopPropagation()}
            >
              {task.title}
            </a>
          </Table.Cell>
          <Table.Cell>
            <span
              class="inline-flex items-center rounded-md border bg-clip-padding px-2 py-0.5 text-xs font-semibold {priorityStyles[
                task.priority
              ] || 'border-gray-200 text-gray-400'}"
            >
              {task.priority}
            </span>
          </Table.Cell>
          <Table.Cell>
            <span
              class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-semibold {statusStyles[
                task.status
              ] || 'bg-gray-100 text-gray-500'}"
            >
              {task.status}
            </span>
          </Table.Cell>
          <Table.Cell>
            <span
              class="inline-flex items-center gap-1.5 text-xs {kindText[
                task.kind
              ] || 'text-muted-foreground'}"
            >
              <span
                class="inline-block size-1.5 rounded-full {kindDot[task.kind] ||
                  'bg-muted-foreground'}"
              ></span>
              {task.kind}
            </span>
          </Table.Cell>
          <Table.Cell class="text-muted-foreground text-xs">
            {task.assignee?.name || "—"}
          </Table.Cell>
          <Table.Cell class="text-muted-foreground text-xs">
            {formatDate(task.due_date)}
          </Table.Cell>
          <Table.Cell class="text-muted-foreground text-xs">
            {formatDate(task.created_at)}
          </Table.Cell>
        </Table.Row>
      {/each}
    {/if}
  </Table.Body>
</Table.Root>
