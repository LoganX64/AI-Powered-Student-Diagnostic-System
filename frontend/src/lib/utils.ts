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
