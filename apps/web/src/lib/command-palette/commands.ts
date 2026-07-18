import type { SearchResult } from "@/types/search";

export type CommandContext = {
  currentTripId?: string | null;
  canEditCurrentTrip?: boolean;
  isOpsAdmin?: boolean;
};

export type CommandPaletteCommand = {
  id: string;
  title: string;
  description: string;
  category: string;
  href?: string;
  shortcut?: string;
  requiredPermission?: string;
  visibleIf: (context: CommandContext) => boolean;
  disabledReason?: (context: CommandContext) => string | undefined;
};

type Translator = (key: string) => string;

export function getCommandRegistry(t: Translator): CommandPaletteCommand[] {
  return [
    {
      id: "onboarding.gettingStarted",
      title: t("commands.gettingStarted.title"),
      description: t("commands.gettingStarted.description"),
      category: t("categories.onboarding"),
      href: "/getting-started",
      visibleIf: () => true
    },
    {
      id: "onboarding.createFirstTrip",
      title: t("commands.createFirstTrip.title"),
      description: t("commands.createFirstTrip.description"),
      category: t("categories.onboarding"),
      href: "/trips/new?mode=destination",
      visibleIf: () => true
    },
    {
      id: "onboarding.discovery",
      title: t("commands.helpChooseDestination.title"),
      description: t("commands.helpChooseDestination.description"),
      category: t("categories.onboarding"),
      href: "/trips/new?mode=discovery",
      visibleIf: () => true
    },
    {
      id: "onboarding.demoTrip",
      title: t("commands.demoTrip.title"),
      description: t("commands.demoTrip.description"),
      category: t("categories.onboarding"),
      href: "/demo-trip",
      visibleIf: () => true
    },
    {
      id: "onboarding.restart",
      title: t("commands.restartOnboarding.title"),
      description: t("commands.restartOnboarding.description"),
      category: t("categories.onboarding"),
      href: "/getting-started?restart=true",
      visibleIf: () => true
    },
    {
      id: "trip.create",
      title: t("commands.createTrip.title"),
      description: t("commands.createTrip.description"),
      category: t("categories.trips"),
      href: "/trips/new",
      visibleIf: () => true
    },
    {
      id: "trip.list",
      title: t("commands.openTrips.title"),
      description: t("commands.openTrips.description"),
      category: t("categories.trips"),
      href: "/trips",
      visibleIf: () => true
    },
    {
      id: "library.open",
      title: t("commands.travelLibrary.title"),
      description: t("commands.travelLibrary.description"),
      category: t("categories.trips"),
      href: "/library",
      visibleIf: () => true
    },
    {
      id: "library.archived",
      title: t("commands.archivedTrips.title"),
      description: t("commands.archivedTrips.description"),
      category: t("categories.trips"),
      href: "/library?lifecycle=archived",
      visibleIf: () => true
    },
    tripSectionCommand("trip.commandCenter", "commandCenter", "command_center", t),
    {
      id: "trip.recap",
      title: t("commands.recap.title"),
      description: t("commands.recap.description"),
      category: t("categories.tripSections"),
      href: "__CURRENT_TRIP__/recap",
      visibleIf: (context) => Boolean(context.currentTripId)
    },
    {
      id: "trip.travelDay",
      title: t("commands.travelDay.title"),
      description: t("commands.travelDay.description"),
      category: t("categories.trips"),
      href: "__CURRENT_TRIP__/today",
      visibleIf: (context) => Boolean(context.currentTripId)
    },
    tripSectionCommand("trip.health", "tripHealth", "health", t),
    tripSectionCommand("trip.route", "routeTransport", "route", t),
    tripSectionCommand("trip.budget", "budget", "budget", t),
    tripSectionCommand("trip.expenses", "expenses", "expenses", t),
    {
      id: "trip.uploadReceipt",
      title: t("commands.uploadReceipt.title"),
      description: t("commands.uploadReceipt.description"),
      category: t("categories.money"),
      href: "__CURRENT_TRIP__?tab=receipts&action=upload",
      requiredPermission: "expenses:edit",
      visibleIf: (context) => Boolean(context.currentTripId),
      disabledReason: (context) =>
        context.canEditCurrentTrip ? undefined : t("disabled.editTrip")
    },
    {
      id: "trip.addExpense",
      title: t("commands.addExpense.title"),
      description: t("commands.addExpense.description"),
      category: t("categories.money"),
      href: "__CURRENT_TRIP__?tab=expenses&action=add",
      requiredPermission: "expenses:edit",
      visibleIf: (context) => Boolean(context.currentTripId),
      disabledReason: (context) =>
        context.canEditCurrentTrip ? undefined : t("disabled.editTrip")
    },
    tripSectionCommand("trip.checklist", "checklist", "checklist", t),
    tripSectionCommand("trip.reminders", "reminders", "reminders", t),
    {
      id: "notifications.open",
      title: t("commands.notifications.title"),
      description: t("commands.notifications.description"),
      category: t("categories.notifications"),
      href: "/notifications",
      visibleIf: () => true
    },
    {
      id: "settings.notifications",
      title: t("commands.notificationSettings.title"),
      description: t("commands.notificationSettings.description"),
      category: t("categories.settings"),
      href: "/settings#notifications",
      visibleIf: () => true
    },
    {
      id: "settings.profile",
      title: t("commands.profileSettings.title"),
      description: t("commands.profileSettings.description"),
      category: t("categories.settings"),
      href: "/settings",
      visibleIf: () => true
    },
    {
      id: "offline.trips",
      title: t("commands.offlineTrips.title"),
      description: t("commands.offlineTrips.description"),
      category: t("categories.offline"),
      href: "/offline-trips",
      visibleIf: () => true
    },
    {
      id: "templates.open",
      title: t("commands.templates.title"),
      description: t("commands.templates.description"),
      category: t("categories.templates"),
      href: "/templates",
      visibleIf: () => true
    },
    {
      id: "workspaces.switcher",
      title: t("commands.workspaces.title"),
      description: t("commands.workspaces.description"),
      category: t("categories.workspaces"),
      href: "/workspaces",
      visibleIf: () => true
    },
    {
      id: "ops.dashboard",
      title: t("commands.opsDashboard.title"),
      description: t("commands.opsDashboard.description"),
      category: t("categories.ops"),
      href: "/ops",
      requiredPermission: "ops:admin",
      visibleIf: (context) => Boolean(context.isOpsAdmin)
    },
    {
      id: "ops.aiGenerations",
      title: t("commands.aiGenerations.title"),
      description: t("commands.aiGenerations.description"),
      category: t("categories.ops"),
      href: "/ops/ai-generations",
      requiredPermission: "ops:admin",
      visibleIf: (context) => Boolean(context.isOpsAdmin)
    }
  ];
}

export function resolveCommandHref(command: CommandPaletteCommand, context: CommandContext) {
  if (!command.href) {
    return undefined;
  }
  if (command.href.startsWith("__CURRENT_TRIP__")) {
    if (!context.currentTripId) {
      return undefined;
    }
    return command.href.replace("__CURRENT_TRIP__", `/trips/${context.currentTripId}`);
  }
  return command.href;
}

export function commandToResult(
  command: CommandPaletteCommand,
  context: CommandContext
): SearchResult | null {
  const href = resolveCommandHref(command, context);
  const disabledReason = command.disabledReason?.(context);
  if (!href || disabledReason) {
    return null;
  }
  return {
    id: `command:${command.id}`,
    type: command.id.startsWith("ops.") ? "ops_page" : "command",
    title: command.title,
    description: command.description,
    href,
    icon: "command",
    category: command.category,
    score: 1,
    metadata: {
      commandId: command.id,
      shortcut: command.shortcut,
      requiredPermission: command.requiredPermission
    }
  };
}

export function filterCommands(
  commands: CommandPaletteCommand[],
  query: string,
  context: CommandContext
) {
  const normalized = query.trim().toLowerCase();
  return commands
    .filter((command) => command.visibleIf(context))
    .filter((command) => {
      if (!normalized) {
        return true;
      }
      return `${command.title} ${command.description} ${command.category}`
        .toLowerCase()
        .includes(normalized);
    });
}

function tripSectionCommand(
  id: string,
  key: string,
  tab: string,
  t: Translator
): CommandPaletteCommand {
  return {
    id,
    title: t(`commands.${key}.title`),
    description: t(`commands.${key}.description`),
    category: t("categories.tripSections"),
    href: `__CURRENT_TRIP__?tab=${tab}`,
    visibleIf: (context) => Boolean(context.currentTripId)
  };
}
