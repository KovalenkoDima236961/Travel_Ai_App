import Link from "next/link";
import type { ReactNode } from "react";

type StartOptionCardProps = {
  href: string;
  title: string;
  description: string;
  bestFor: string;
  estimatedTime: string;
  icon?: ReactNode;
  onSelect?: () => void;
};

export function StartOptionCard({
  href,
  title,
  description,
  bestFor,
  estimatedTime,
  icon,
  onSelect
}: StartOptionCardProps) {
  return (
    <Link
      href={href}
      onClick={onSelect}
      aria-label={`${title}. ${estimatedTime}`}
      className="group flex h-full flex-col rounded-[18px] border border-sand-300 bg-white p-5 transition hover:-translate-y-0.5 hover:border-sand-500 hover:shadow-[0_12px_28px_rgba(34,26,20,0.07)] focus:outline-none focus:ring-2 focus:ring-clay/30"
    >
      {icon ? (
        <span className="flex h-10 w-10 items-center justify-center rounded-full bg-clay-tint text-clay-dark">
          {icon}
        </span>
      ) : null}
      <h3 className="mt-4 text-[16px] font-semibold text-cocoa-900">{title}</h3>
      <p className="mt-2 text-[13.5px] leading-[1.55] text-cocoa-500">{description}</p>
      <p className="mt-4 text-[12.5px] text-cocoa-400">
        <span className="font-semibold text-cocoa-600">{bestFor}</span>
      </p>
      <p className="mt-auto pt-3 text-[12.5px] font-semibold text-[#3E6B5A]">{estimatedTime}</p>
    </Link>
  );
}
