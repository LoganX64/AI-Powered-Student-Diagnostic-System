import { useEffect, useState, useRef } from "react";
import { ChevronDownIcon, CheckIcon, SearchIcon } from "lucide-react";

export type SearchableSelectOption = {
  label: string;
  value: string;
  search?: string;
};

type SearchableSelectProps = {
  options: SearchableSelectOption[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  cap?: number;
};

export function SearchableSelect({
  options,
  value,
  onChange,
  placeholder = "Search...",
  disabled = false,
  cap = 50,
}: SearchableSelectProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);

  const selected = options.find((o) => o.value === value);

  const displayValue = open ? search : selected?.label ?? "";

  const filtered = options.filter((o) => {
    const term = search.toLowerCase();
    return (
      o.label.toLowerCase().includes(term) ||
      (o.search && o.search.toLowerCase().includes(term))
    );
  });

  const totalMatches = filtered.length;
  const displayed = filtered.slice(0, cap);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setSearch("");
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div className="relative" ref={containerRef}>
      <div className="relative">
        <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <input
          type="text"
          placeholder={placeholder}
          value={displayValue}
          onChange={(e) => {
            setSearch(e.target.value);
            if (!open) setOpen(true);
          }}
          onFocus={() => {
            setOpen(true);
            setSearch("");
          }}
          disabled={disabled}
          className="flex h-9 w-full rounded-md border bg-transparent pl-9 pr-8 py-2 text-sm transition-colors placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
        />
        <ChevronDownIcon className="absolute right-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
      </div>

      {open && (
        <div className="absolute z-50 mt-1 max-h-60 w-full overflow-y-auto rounded-md border bg-popover p-1 shadow-sm">
          {displayed.length === 0 ? (
            <p className="py-2 text-center text-xs text-muted-foreground">No results found.</p>
          ) : (
            displayed.map((o) => (
              <button
                key={o.value}
                type="button"
                onClick={() => {
                  onChange(o.value);
                  setOpen(false);
                  setSearch("");
                }}
                className={`flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground ${
                  value === o.value ? "bg-accent" : ""
                }`}
              >
                <CheckIcon className={`size-3.5 shrink-0 ${value === o.value ? "opacity-100" : "opacity-0"}`} />
                <span>{o.label}</span>
              </button>
            ))
          )}
          {totalMatches > cap && (
            <p className="py-1.5 text-center text-xs text-muted-foreground">
              Showing {cap} of {totalMatches} — type more to narrow
            </p>
          )}
        </div>
      )}
    </div>
  );
}
