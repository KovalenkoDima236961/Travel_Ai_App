import type { ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

type ResponsiveDataViewProps<Item> = {
  items: Item[];
  getKey: (item: Item, index: number) => string;
  desktop: ReactNode;
  renderMobileCard: (item: Item, index: number) => ReactNode;
  empty?: ReactNode;
  loading?: ReactNode;
  className?: string;
  mobileClassName?: string;
};

/**
 * Keeps a desktop table available at medium widths and above while exposing the
 * same data as readable, touch-friendly cards on narrow screens.
 */
export function ResponsiveDataView<Item>({
  items,
  getKey,
  desktop,
  renderMobileCard,
  empty,
  loading,
  className,
  mobileClassName
}: ResponsiveDataViewProps<Item>) {
  if (loading) {
    return <>{loading}</>;
  }

  if (items.length === 0) {
    return <>{empty ?? null}</>;
  }

  return (
    <div className={cn("min-w-0", className)}>
      <div className="hidden md:block">{desktop}</div>
      <ul className={cn("space-y-3 md:hidden", mobileClassName)}>
        {items.map((item, index) => (
          <li key={getKey(item, index)}>{renderMobileCard(item, index)}</li>
        ))}
      </ul>
    </div>
  );
}
