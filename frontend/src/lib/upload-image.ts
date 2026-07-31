import { initUpload, confirmUpload } from "$lib/api/attachments";

const ALLOWED_TYPES = new Set([
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
]);
const MAX_SIZE = 100 * 1024 * 1024;
// Presigned URLs expire after 15 minutes (backend default), so the upload
// timeout must stay below that. 10 minutes leaves room for init/confirm calls.
const UPLOAD_TIMEOUT_MAX_MS = 10 * 60_000;
const UPLOAD_TIMEOUT_MIN_MS = 60_000;
const MIN_UPSTREAM_BYTES_PER_SEC = 256 * 1024;

export interface UploadResult {
  attachmentId: number;
  fileName: string;
}

export async function uploadInlineImage(
  taskId: number,
  file: File,
  options?: { onProgress?: (percent: number) => void },
): Promise<UploadResult> {
  if (!ALLOWED_TYPES.has(file.type)) {
    throw new Error(
      `Invalid file type "${file.type}". Allowed: PNG, JPEG, GIF, WebP.`,
    );
  }

  if (file.size > MAX_SIZE) {
    throw new Error("File exceeds 100 MB limit");
  }

  const init = await initUpload(taskId, {
    file_name: file.name,
    size_bytes: file.size,
  });

  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.timeout = Math.min(
      UPLOAD_TIMEOUT_MAX_MS,
      Math.max(UPLOAD_TIMEOUT_MIN_MS, (file.size / MIN_UPSTREAM_BYTES_PER_SEC) * 1000),
    );
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && options?.onProgress) {
        options.onProgress(Math.round((e.loaded / e.total) * 100));
      }
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) resolve();
      else reject(new Error(`Upload failed with status ${xhr.status}`));
    };
    xhr.onerror = () =>
      reject(new Error("Upload failed: network error while uploading to storage"));
    xhr.ontimeout = () => reject(new Error("Upload timed out"));
    xhr.onabort = () => reject(new Error("Upload was canceled"));
    xhr.open("PUT", init.upload_url);
    xhr.send(file);
  });

  const confirmed = await confirmUpload(taskId, init.id);

  return { attachmentId: confirmed.id, fileName: confirmed.file_name };
}
