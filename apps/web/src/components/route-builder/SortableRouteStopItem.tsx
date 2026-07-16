import { useTranslations } from "next-intl";
import type { DragEvent } from "react";
import type { TripRouteStop } from "@/entities/route/model";

type SortableRouteStopItemProps = {
  stop: TripRouteStop;
  index: number;
  canMoveUp: boolean;
  canMoveDown: boolean;
  disabled?: boolean;
  dragging?: boolean;
  onDragStart: (event: DragEvent<HTMLLIElement>, stopId: string) => void;
  onDragOver: (event: DragEvent<HTMLLIElement>, stopId: string) => void;
  onDrop: (event: DragEvent<HTMLLIElement>, stopId: string) => void;
  onDragEnd: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
};

export function SortableRouteStopItem({
  stop,
  index,
  canMoveUp,
  canMoveDown,
  disabled = false,
  dragging = false,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
  onMoveUp,
  onMoveDown
}: SortableRouteStopItemProps) {
  const t = useTranslations("route");
  const name = stop.city || stop.destination || t("unnamedStop");
  return (
    <li
      aria-roledescription={t("sortableStop")}
      className={`flex items-center gap-3 rounded-[14px] border bg-white p-3 transition ${
        dragging ? "border-clay opacity-60 shadow-md" : "border-sand-300"
      }`}
      draggable={!disabled}
      onDragEnd={onDragEnd}
      onDragOver={(event) => onDragOver(event, stop.id)}
      onDragStart={(event) => onDragStart(event, stop.id)}
      onDrop={(event) => onDrop(event, stop.id)}
    >
      <span
        aria-hidden
        className="flex h-9 w-9 cursor-grab items-center justify-center rounded-lg border border-sand-300 bg-sand-50 text-[18px] text-cocoa-500 active:cursor-grabbing"
      >
        ⠿
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[14px] font-semibold text-cocoa-900">
          {index + 1}. {name}
        </p>
        <p className="truncate text-[12px] text-cocoa-500">
          {[stop.country, stop.arrivalDate].filter(Boolean).join(" · ") || t("datesNotAssigned")}
        </p>
      </div>
      <div className="flex shrink-0 gap-1">
        <button
          aria-label={t("moveStopUp", { name })}
          className="h-9 rounded-lg border border-sand-300 px-2.5 text-[12px] font-semibold text-cocoa-600 transition hover:border-clay focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-clay disabled:cursor-not-allowed disabled:opacity-35"
          disabled={disabled || !canMoveUp}
          onClick={onMoveUp}
          type="button"
        >
          ↑
        </button>
        <button
          aria-label={t("moveStopDown", { name })}
          className="h-9 rounded-lg border border-sand-300 px-2.5 text-[12px] font-semibold text-cocoa-600 transition hover:border-clay focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-clay disabled:cursor-not-allowed disabled:opacity-35"
          disabled={disabled || !canMoveDown}
          onClick={onMoveDown}
          type="button"
        >
          ↓
        </button>
      </div>
    </li>
  );
}
