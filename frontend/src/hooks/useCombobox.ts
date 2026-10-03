import { useCallback, useId, useRef, useState } from "react";

/**
 * Keyboard and ARIA wiring shared by the hand-rolled comboboxes in this app
 * (SearchableSelect, StudentPicker, and the coach/subject pickers on the student
 * and test forms).
 *
 * Follows the WAI-ARIA 1.2 combobox pattern with `aria-activedescendant` rather
 * than moving DOM focus into the listbox, so the text input keeps focus and
 * caret position while arrowing through options.
 *
 * `items` is the already-filtered list; this hook only tracks which one is
 * highlighted and moves the selection.
 */
export function useCombobox<T>(items: T[], onSelect: (item: T) => void) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);
  const listboxId = useId();
  const activeId = activeIndex >= 0 ? `${listboxId}-option-${activeIndex}` : undefined;

  const openList = useCallback(() => {
    setOpen(true);
    setActiveIndex(items.length > 0 ? 0 : -1);
  }, [items.length]);

  const closeList = useCallback(() => {
    setOpen(false);
    setActiveIndex(-1);
  }, []);

  const commit = useCallback(
    (index: number) => {
      const item = items[index];
      if (!item) return;
      onSelect(item);
      closeList();
    },
    [items, onSelect, closeList],
  );

  /**
   * Wires ArrowDown/ArrowUp/Home/End/Enter/Escape/Tab onto the trigger input.
   * Tab and Escape close without committing so the form still validates.
   */
  const inputProps = (disabled = false) => ({
    role: "combobox" as const,
    "aria-expanded": open,
    "aria-controls": open ? listboxId : undefined,
    "aria-autocomplete": "list" as const,
    "aria-activedescendant": open ? activeId : undefined,
    disabled,
    onKeyDown: (e: React.KeyboardEvent) => {
      if (disabled) return;
      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          if (!open) {
            openList();
          } else {
            setActiveIndex((i) => (items.length ? (i + 1) % items.length : -1));
          }
          break;
        case "ArrowUp":
          e.preventDefault();
          if (!open) {
            openList();
          } else {
            setActiveIndex((i) =>
              items.length ? (i - 1 + items.length) % items.length : -1
            );
          }
          break;
        case "Home":
          if (open) {
            e.preventDefault();
            setActiveIndex(items.length ? 0 : -1);
          }
          break;
        case "End":
          if (open) {
            e.preventDefault();
            setActiveIndex(items.length - 1);
          }
          break;
        case "Enter":
          // Only intercept Enter while a row is highlighted, so the form can
          // still submit normally when the list is closed.
          if (open && activeIndex >= 0) {
            e.preventDefault();
            commit(activeIndex);
          }
          break;
        case "Escape":
          if (open) {
            e.preventDefault();
            closeList();
          }
          break;
        case "Tab":
          closeList();
          break;
      }
    },
  });

  /** Props for the popup container. */
  const listboxProps = {
    id: listboxId,
    role: "listbox" as const,
  };

  /** Props for one row. `index` must match its position in `items`. */
  const optionProps = (index: number) => ({
    id: `${listboxId}-option-${index}`,
    role: "option" as const,
    "aria-selected": index === activeIndex,
  });

  return {
    open,
    setOpen,
    openList,
    closeList,
    activeIndex,
    setActiveIndex,
    containerRef,
    inputProps,
    listboxProps,
    optionProps,
  };
}