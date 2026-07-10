package workspacepolicies

func DefaultRules() RulesDocument {
	return RulesDocument{
		SchemaVersion: SchemaVersion,
		Rules: Rules{
			RequireTripBudget: Rule{Enabled: false, Severity: SeverityWarning},
			MaxTripBudget: MoneyRule{
				Rule: Rule{Enabled: false, Severity: SeverityBlocking}, Currency: "EUR",
			},
			MaxDailyBudget: MoneyRule{
				Rule: Rule{Enabled: false, Severity: SeverityWarning}, Currency: "EUR",
			},
			MaxItemCost: ItemCostRule{
				MoneyRule: MoneyRule{
					Rule: Rule{Enabled: false, Severity: SeverityWarning}, Currency: "EUR",
				},
				Categories: []string{},
			},
			MaxAccommodationTotal: MoneyRule{
				Rule: Rule{Enabled: false, Severity: SeverityWarning}, Currency: "EUR",
			},
			MaxAccommodationPerNight: MoneyRule{
				Rule: Rule{Enabled: false, Severity: SeverityWarning}, Currency: "EUR",
			},
			RequireCostSplitting:                Rule{Enabled: false, Severity: SeverityWarning},
			RequireAvailabilityForTicketedItems: Rule{Enabled: true, Severity: SeverityWarning},
			MaxWalkingKmPerDay: WalkingRule{
				Rule: Rule{Enabled: true, Severity: SeverityWarning}, Km: 12,
			},
			NoLateActivitiesAfter: LateActivityRule{
				Rule: Rule{Enabled: true, Severity: SeverityWarning}, Time: "22:00",
			},
			RequiredRestTimePerDay: RestTimeRule{
				Rule: Rule{Enabled: false, Severity: SeverityInfo}, Minutes: 60,
			},
			PreferredTransportModes: TransportRule{
				Rule: Rule{Enabled: false, Severity: SeverityInfo}, Modes: []string{},
			},
			MaxTransferHoursPerDay: TransferHoursRule{
				Rule: Rule{Enabled: false, Severity: SeverityWarning}, Hours: 6,
			},
			DisallowedTransportModes: TransportRule{
				Rule: Rule{Enabled: false, Severity: SeverityWarning}, Modes: []string{},
			},
			DisallowedActivityTypes: ActivityTypesRule{
				Rule: Rule{Enabled: false, Severity: SeverityWarning}, Types: []string{},
			},
		},
	}
}
