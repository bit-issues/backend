import { uploadInlineImage, type UploadResult } from "$lib/upload-image";
import { toast } from "$lib/toast";

export function attachmentMarkdown(result: UploadResult): string {
  const name = result.fileName
    .replace(/\.[^.]+$/, "")
    .replace(/[[\]\\]/g, "");
  return `![${name}](attachment:${result.attachmentId})`;
}

export function insertAtCursor(
  el: HTMLTextAreaElement | undefined,
  current: string,
  insertion: string,
  setValue: (next: string) => void,
): void {
  if (!el) {
    setValue(current + insertion);
    return;
  }
  const start = el.selectionStart ?? current.length;
  const end = el.selectionEnd ?? start;
  setValue(current.slice(0, start) + insertion + current.slice(end));
  requestAnimationFrame(() => {
    const pos = start + insertion.length;
    el.selectionStart = pos;
    el.selectionEnd = pos;
    el.focus();
  });
}

export type InlineImageEditorOptions = {
  getTaskId: () => number;
  getValue: () => string;
  setValue: (value: string) => void;
  getTextarea: () => HTMLTextAreaElement | undefined;
  getSessionId?: () => unknown;
};

export class InlineImageEditor {
  uploading = $state(false);

  #opts: InlineImageEditorOptions;

  constructor(opts: InlineImageEditorOptions) {
    this.#opts = opts;
  }

  async uploadAndInsert(file: File): Promise<void> {
    const taskId = this.#opts.getTaskId();
    if (!taskId) return;
    if (this.uploading) {
      toast("Image upload already in progress");
      return;
    }

    const sessionId = this.#opts.getSessionId?.();
    this.uploading = true;
    try {
      const result = await uploadInlineImage(taskId, file);
      if (
        this.#opts.getSessionId &&
        this.#opts.getSessionId() !== sessionId
      ) {
        toast("Image uploaded, but the comment is no longer being edited");
        return;
      }
      insertAtCursor(
        this.#opts.getTextarea(),
        this.#opts.getValue(),
        attachmentMarkdown(result),
        this.#opts.setValue,
      );
      toast.success("Image uploaded");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Image upload failed");
    } finally {
      this.uploading = false;
    }
  }

  handleImagePick(e: Event): void {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (file) void this.uploadAndInsert(file);
    input.value = "";
  }

  handleImagePaste(e: ClipboardEvent): void {
    if (!this.#opts.getTaskId()) return;
    const items = e.clipboardData?.items;
    if (!items) return;
    for (const item of items) {
      if (item.kind === "file" && item.type.startsWith("image/")) {
        e.preventDefault();
        const file = item.getAsFile();
        if (file) void this.uploadAndInsert(file);
        return;
      }
    }
  }

  async handleImageDrop(e: DragEvent): Promise<void> {
    if (!this.#opts.getTaskId()) return;
    e.preventDefault();
    const files = e.dataTransfer?.files;
    if (!files) return;
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      if (file.type.startsWith("image/")) {
        await this.uploadAndInsert(file);
      } else {
        toast.error(
          `"${file.name}" is not an image. Only images can be dropped.`,
        );
      }
    }
  }

  handleDragOver(e: DragEvent): void {
    if (!this.#opts.getTaskId()) return;
    e.preventDefault();
  }
}
