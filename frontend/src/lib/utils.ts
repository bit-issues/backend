import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export type WithElementRef<T> = T & {
  ref?: any;
  children?: any;
};

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
