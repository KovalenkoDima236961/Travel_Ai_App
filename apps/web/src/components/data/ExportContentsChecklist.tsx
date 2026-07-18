"use client";

import type { AccountExportSections } from "@/types/data-export";

const labels: Array<[keyof AccountExportSections, string]> = [
  ["profile", "Profile"], ["preferences", "Preferences"], ["personalTrips", "Trips"],
  ["tripRecaps", "Trip recaps"], ["templates", "Templates"], ["expenses", "Expenses"],
  ["settlements", "Settlements"], ["checklists", "Checklists"], ["reminders", "Reminders"],
  ["personalizationFeedback", "Personalization feedback"], ["notificationPreferences", "Notification preferences"], ["notifications", "Notifications"]
];

export function ExportContentsChecklist({ sections, onChange }: { sections: AccountExportSections; onChange: (next: AccountExportSections) => void }) {
  return (
    <fieldset className="mt-5 grid gap-2 sm:grid-cols-2">
      <legend className="text-sm font-semibold text-cocoa-700">Choose export contents</legend>
      {labels.map(([key, label]) => (
        <label className="flex items-center gap-2 text-sm text-cocoa-600" key={key}>
          <input checked={sections[key]} onChange={(event) => onChange({ ...sections, [key]: event.target.checked })} type="checkbox" />
          {label}
        </label>
      ))}
    </fieldset>
  );
}
