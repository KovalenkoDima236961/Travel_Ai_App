DROP INDEX IF EXISTS idx_feature_flag_audit_events_created;
DROP INDEX IF EXISTS idx_feature_flag_audit_events_flag_created;
DROP TABLE IF EXISTS feature_flag_audit_events;

DROP INDEX IF EXISTS idx_feature_flags_environment_scope;
DROP INDEX IF EXISTS idx_feature_flags_lookup;
DROP INDEX IF EXISTS idx_feature_flags_scope_unique;
DROP TABLE IF EXISTS feature_flags;
