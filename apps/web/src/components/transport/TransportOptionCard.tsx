import type { TransportOption } from "@/types/transport";
import { TransportOptionMiniCard } from "./TransportOptionMiniCard";

type Props = {
  option: TransportOption;
  disabled?: boolean;
  selecting?: boolean;
  onSelect: (option: TransportOption) => void;
};

export function TransportOptionCard({ option, disabled = false, selecting = false, onSelect }: Props) {
  return (
    <TransportOptionMiniCard
      disabled={disabled}
      onSelect={onSelect}
      option={option}
      selecting={selecting}
    />
  );
}
