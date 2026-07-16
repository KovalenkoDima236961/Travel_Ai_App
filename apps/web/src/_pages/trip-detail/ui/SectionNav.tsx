import Link from "next/link";
import type { ComponentType } from "react";
import { buildNavigationGroups } from "@/lib/trip-command-center/navigation";
import type { NavigationGroup } from "@/types/trip-command-center";
import {
  CalendarIcon,
  ChartBarIcon,
  MapIcon,
  UserGroupIcon,
  WalletIcon
} from "./icons";

type IconComponent = ComponentType<{ className?: string }>;

const GROUP_ICONS: Record<NavigationGroup["id"], IconComponent> = {
  plan: CalendarIcon,
  prepare: MapIcon,
  money: WalletIcon,
  team: UserGroupIcon,
  control: ChartBarIcon
};

/**
 * In-page section rail for the Trip Detail screen. Links jump to the anchored
 * regions below; Analytics routes to the (still slate) analytics screen.
 */
export function SectionNav({
  tripId,
  navigationGroups
}: {
  tripId: string;
  navigationGroups?: NavigationGroup[];
}) {
  const groups = navigationGroups ?? buildNavigationGroups({ tripId });
  return (
    <nav className="flex flex-col gap-5" aria-label="Trip sections">
      {groups.map((group) => {
        const GroupIcon = GROUP_ICONS[group.id] ?? ChartBarIcon;
        return (
          <div key={group.id}>
            <h2 className="mb-1 flex items-center gap-2 px-3.5 text-[11px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              <GroupIcon className="h-[14px] w-[14px]" />
              {group.label}
            </h2>
            <div className="flex flex-col gap-0.5">
              {group.items.map((item) => (
                <NavLink
                  key={`${group.id}:${item.id}:${item.href}`}
                  active={item.id === "overview" && item.label === "Overview"}
                  badge={item.badge}
                  href={item.href}
                  label={item.label}
                />
              ))}
            </div>
          </div>
        );
      })}
    </nav>
  );
}

function NavLink({
  active,
  badge,
  href,
  label
}: {
  active: boolean;
  badge?: number | string | null;
  href: string;
  label: string;
}) {
  const className = active
    ? "flex items-center justify-between gap-3 rounded-xl bg-sand-200 px-3.5 py-2.5 text-[14px] font-semibold text-cocoa-900"
    : "flex items-center justify-between gap-3 rounded-xl px-3.5 py-2.5 text-[14px] font-medium text-cocoa-500 transition hover:bg-sand-200 hover:text-cocoa-900";
  const content = (
    <>
      <span className="truncate">{label}</span>
      {badge ? (
        <span className="shrink-0 rounded-full bg-[#FBF0EB] px-2 py-0.5 text-[11px] font-semibold text-[#A93624]">
          {badge}
        </span>
      ) : null}
    </>
  );
  if (href.startsWith("/")) {
    return (
      <Link href={href} className={className}>
        {content}
      </Link>
    );
  }
  return (
    <a href={href} className={className}>
      {content}
    </a>
  );
}
