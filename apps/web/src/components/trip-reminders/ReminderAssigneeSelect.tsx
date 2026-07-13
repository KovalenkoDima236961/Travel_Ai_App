import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";

type ReminderAssigneeSelectProps = {
  value: string;
  currentUserId?: string | null;
  disabled?: boolean;
  label: string;
  placeholder: string;
  assignMeLabel: string;
  onChange: (value: string) => void;
};

export function ReminderAssigneeSelect({
  value,
  currentUserId,
  disabled,
  label,
  placeholder,
  assignMeLabel,
  onChange
}: ReminderAssigneeSelectProps) {
  return (
    <label className="text-sm font-medium text-slate-700">
      {label}
      <div className="mt-1 grid gap-2 sm:grid-cols-[1fr_auto]">
        <Input
          disabled={disabled}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
          value={value}
        />
        {currentUserId ? (
          <Button
            disabled={disabled}
            onClick={() => onChange(currentUserId)}
            type="button"
            variant="secondary"
          >
            {assignMeLabel}
          </Button>
        ) : null}
      </div>
    </label>
  );
}
