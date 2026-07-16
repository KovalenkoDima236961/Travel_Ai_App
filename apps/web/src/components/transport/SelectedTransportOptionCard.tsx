import type { SelectedTransportOption } from "@/types/transport";
import { SelectedTransportSummary } from "./SelectedTransportSummary";

type Props = {
  option?: SelectedTransportOption | null;
  stale?: boolean;
  canEdit?: boolean;
  removing?: boolean;
  onRemove?: () => void;
};

export function SelectedTransportOptionCard({ option, stale = false, canEdit = false, removing = false, onRemove }: Props) {
  return (
    <SelectedTransportSummary
      canRemove={canEdit}
      onRemove={onRemove}
      option={option}
      removing={removing}
      stale={stale}
    />
  );
}
