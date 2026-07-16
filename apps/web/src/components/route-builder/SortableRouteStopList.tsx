"use client";

import { useState, type DragEvent } from "react";
import { useTranslations } from "next-intl";
import type { TripRouteStop } from "@/entities/route/model";
import { SortableRouteStopItem } from "./SortableRouteStopItem";

type SortableRouteStopListProps = {
  stops: TripRouteStop[];
  disabled?: boolean;
  onReorder: (fromIndex: number, toIndex: number) => void;
};

export function SortableRouteStopList({
  stops,
  disabled = false,
  onReorder
}: SortableRouteStopListProps) {
  const t = useTranslations("route");
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [announcement, setAnnouncement] = useState("");

  function move(fromIndex: number, toIndex: number) {
    if (disabled || fromIndex === toIndex || toIndex < 0 || toIndex >= stops.length) {
      return;
    }
    const name = stops[fromIndex]?.city || stops[fromIndex]?.destination || t("unnamedStop");
    onReorder(fromIndex, toIndex);
    setAnnouncement(t("movedStopAnnouncement", { name, position: toIndex + 1 }));
  }

  function handleDragStart(event: DragEvent<HTMLLIElement>, stopId: string) {
    if (disabled) {
      event.preventDefault();
      return;
    }
    setDraggingId(stopId);
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", stopId);
  }

  function handleDragOver(event: DragEvent<HTMLLIElement>) {
    if (!disabled) {
      event.preventDefault();
      event.dataTransfer.dropEffect = "move";
    }
  }

  function handleDrop(event: DragEvent<HTMLLIElement>, targetId: string) {
    event.preventDefault();
    const sourceId = event.dataTransfer.getData("text/plain") || draggingId;
    const fromIndex = stops.findIndex((stop) => stop.id === sourceId);
    const toIndex = stops.findIndex((stop) => stop.id === targetId);
    move(fromIndex, toIndex);
    setDraggingId(null);
  }

  return (
    <section aria-label={t("reorderStops")}>
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <p className="text-[13px] font-semibold text-cocoa-800">{t("reorderStops")}</p>
        <p className="text-[12px] text-cocoa-500">{t("dragOrButtonsHint")}</p>
      </div>
      <ol className="space-y-2">
        {stops.map((stop, index) => (
          <SortableRouteStopItem
            canMoveDown={index < stops.length - 1}
            canMoveUp={index > 0}
            disabled={disabled}
            dragging={draggingId === stop.id}
            index={index}
            key={stop.id}
            onDragEnd={() => setDraggingId(null)}
            onDragOver={handleDragOver}
            onDragStart={handleDragStart}
            onDrop={handleDrop}
            onMoveDown={() => move(index, index + 1)}
            onMoveUp={() => move(index, index - 1)}
            stop={stop}
          />
        ))}
      </ol>
      <p aria-live="polite" className="sr-only">{announcement}</p>
    </section>
  );
}
