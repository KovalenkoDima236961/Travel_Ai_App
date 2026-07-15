package aivalidation

type Config struct {
	Enabled                    bool
	RepairEnabled              bool
	MaxRepairAttempts          int
	BlockOnSchemaErrors        bool
	BlockOnPolicyBlockers      bool
	BlockOnCriticalRouteErrors bool
	BlockOnBudgetErrors        bool
	FailOpen                   bool
}

func DefaultConfig() Config {
	return Config{
		Enabled:                    true,
		RepairEnabled:              true,
		MaxRepairAttempts:          2,
		BlockOnSchemaErrors:        true,
		BlockOnPolicyBlockers:      true,
		BlockOnCriticalRouteErrors: true,
		BlockOnBudgetErrors:        true,
		FailOpen:                   false,
	}
}

func NormalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()
	if !cfg.Enabled {
		defaults.Enabled = false
	}
	defaults.RepairEnabled = cfg.RepairEnabled
	if cfg.MaxRepairAttempts > 0 {
		defaults.MaxRepairAttempts = cfg.MaxRepairAttempts
	}
	defaults.BlockOnSchemaErrors = cfg.BlockOnSchemaErrors
	defaults.BlockOnPolicyBlockers = cfg.BlockOnPolicyBlockers
	defaults.BlockOnCriticalRouteErrors = cfg.BlockOnCriticalRouteErrors
	defaults.BlockOnBudgetErrors = cfg.BlockOnBudgetErrors
	defaults.FailOpen = cfg.FailOpen
	return defaults
}
