export type ExportStatus = "queued" | "processing" | "completed" | "failed" | "expired";

export type DataExportJob = {
  exportId: string;
  status: ExportStatus;
  createdAt: string;
  fileName?: string;
  sizeBytes?: number;
  checksumSha256?: string;
  expiresAt?: string;
  errorCode?: string;
  errorMessageSafe?: string;
  downloadUrl?: string;
};

export type AccountExportSections = {
  profile: boolean;
  preferences: boolean;
  personalTrips: boolean;
  tripRecaps: boolean;
  templates: boolean;
  expenses: boolean;
  settlements: boolean;
  checklists: boolean;
  reminders: boolean;
  personalizationFeedback: boolean;
  notificationPreferences: boolean;
  notifications: boolean;
};

export type CreateAccountExportInput = {
  sections: AccountExportSections;
  includeReceiptFiles: boolean;
  includeWorkspaceData: boolean;
};

export type CreateTripArchiveExportInput = {
  includeReceiptFiles: boolean;
  includeRecapPdf: boolean;
  includePrivateNotes: boolean;
};

export type NotificationCleanupInput = {
  olderThanDays: number;
  onlyRead?: boolean;
  categories?: string[];
};

export const DEFAULT_ACCOUNT_EXPORT_SECTIONS: AccountExportSections = {
  profile: true,
  preferences: true,
  personalTrips: true,
  tripRecaps: true,
  templates: true,
  expenses: true,
  settlements: true,
  checklists: true,
  reminders: true,
  personalizationFeedback: true,
  notificationPreferences: true,
  notifications: true
};
