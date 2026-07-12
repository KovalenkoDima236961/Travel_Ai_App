import Link from "next/link";
import type { ComponentType } from "react";
import {
  CalendarIcon,
  ChartBarIcon,
  ChatBubbleIcon,
  MapIcon,
  ScaleIcon,
  UserGroupIcon,
  WalletIcon
} from "./icons";

type IconComponent = ComponentType<{ className?: string }>;

const ITEMS: { label: string; href: string; icon: IconComponent }[] = [
  { label: "Itinerary", href: "#itinerary", icon: CalendarIcon },
  { label: "Dates", href: "#dates", icon: CalendarIcon },
  { label: "Decisions", href: "#decisions", icon: UserGroupIcon },
  { label: "Map", href: "#map", icon: MapIcon },
  { label: "Budget", href: "#budget", icon: WalletIcon },
  { label: "Cost split", href: "#cost-split", icon: ScaleIcon },
  { label: "Sharing", href: "#sharing", icon: UserGroupIcon },
  { label: "Activity", href: "#activity", icon: ChatBubbleIcon }
];

/**
 * In-page section rail for the Trip Detail screen. Links jump to the anchored
 * regions below; Analytics routes to the (still slate) analytics screen.
 */
export function SectionNav({ tripId }: { tripId: string }) {
  return (
    <nav className="flex flex-col gap-0.5">
      {ITEMS.map(({ label, href, icon: IconComponent }, index) => (
        <a
          key={href}
          href={href}
          className={
            index === 0
              ? "flex items-center gap-3 rounded-xl bg-sand-200 px-3.5 py-2.5 text-[14px] font-semibold text-cocoa-900"
              : "flex items-center gap-3 rounded-xl px-3.5 py-2.5 text-[14px] font-medium text-cocoa-500 transition hover:bg-sand-200 hover:text-cocoa-900"
          }
        >
          <IconComponent
            className={index === 0 ? "h-[17px] w-[17px] text-clay" : "h-[17px] w-[17px] text-[#A08D78]"}
          />
          {label}
        </a>
      ))}
      <Link
        href={`/trips/${tripId}/analytics`}
        className="flex items-center gap-3 rounded-xl px-3.5 py-2.5 text-[14px] font-medium text-cocoa-500 transition hover:bg-sand-200 hover:text-cocoa-900"
      >
        <ChartBarIcon className="h-[17px] w-[17px] text-[#A08D78]" />
        Analytics
      </Link>
    </nav>
  );
}
