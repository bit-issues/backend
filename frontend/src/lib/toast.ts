import { toast as sonnerToast } from "svelte-sonner";

interface ToastMessages {
  loading: string;
  success: string;
  error: string;
}

type ToastFn = {
  (msg: string): string | number;
  success(msg: string): string | number;
  error(msg: string): string | number;
  promise<T>(promise: Promise<T>, msgs: ToastMessages): string | number | undefined;
};

const toast: ToastFn = ((msg: string) => sonnerToast(msg)) as ToastFn;
toast.success = (msg: string) => sonnerToast.success(msg);
toast.error = (msg: string) => sonnerToast.error(msg);
toast.promise = <T>(promise: Promise<T>, msgs: ToastMessages) =>
  sonnerToast.promise(promise, msgs);

export { toast };
