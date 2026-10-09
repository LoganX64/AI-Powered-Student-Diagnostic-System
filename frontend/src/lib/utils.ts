import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDateDDMMYYYY(isoOrDate: string): string {
  if (!isoOrDate) return "—";
  const d = new Date(isoOrDate);
  if (isNaN(d.getTime())) return "—";
  const dd = String(d.getDate()).padStart(2, "0");
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const yyyy = d.getFullYear();
  return `${dd}-${mm}-${yyyy}`;
}

export function parseRouteId(id: string | undefined): number | null {
  if (!id) return null;
  const n = Number(id);
  return isNaN(n) ? null : n;
}

export function bytesToGB(bytes: number): string {
  if (bytes === -1) return "Unlimited";
  return `${(bytes / 1073741824).toFixed(1)} GB`;
}

/**
 * Hard ceiling on `limit` for every list endpoint. utils.ParsePagination
 * (`backend/utils/pagination.go`) only honours a limit when it is > 0 and <= 100,
 * and silently falls back to its 50 default otherwise — so asking for 200 returns
 * 50 rows with no error. Reference-data pickers request this instead of a larger
 * number, so the whole list actually arrives.
 */
export const MAX_LIST_LIMIT = 100;

/**
 * Whether a paginated response was cut short, i.e. the server holds more rows
 * than the request asked for. Use it to tell the user the list is partial rather
 * than letting a truncated dropdown look complete.
 */
export function isTruncated(total: number, loaded: number): boolean {
  return total > loaded;
}
