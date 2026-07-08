export function createLatestRequestGuard() {
  let version = 0;

  return {
    next(): number {
      return ++version;
    },
    isLatest(token: number): boolean {
      return token === version;
    },
    invalidate(): void {
      version++;
    },
  };
}

export type LatestRequestGuard = ReturnType<typeof createLatestRequestGuard>;

type RunLatestHandlers<T> = {
  onSuccess: (value: T) => void;
  onError?: (error: unknown) => void;
  onFinally?: () => void;
};

export function runLatest<T>(
  guard: LatestRequestGuard,
  work: () => Promise<T>,
  handlers: RunLatestHandlers<T>,
): void {
  const token = guard.next();
  work()
    .then((value) => {
      if (!guard.isLatest(token)) return;
      handlers.onSuccess(value);
    })
    .catch((error) => {
      if (!guard.isLatest(token)) return;
      handlers.onError?.(error);
    })
    .finally(() => {
      if (!guard.isLatest(token)) return;
      handlers.onFinally?.();
    });
}
