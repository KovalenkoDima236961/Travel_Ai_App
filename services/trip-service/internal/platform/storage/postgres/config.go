package postgres

type Config struct {
	Database             string `yaml:"database" env:"POSTGRES_DB" validate:"required"`
	Username             string `yaml:"username" env:"POSTGRES_USER" validate:"required"`
	Password             string `yaml:"password" env:"POSTGRES_PASSWORD" validate:"required"`
	Host                 string `yaml:"host" env:"POSTGRES_HOST" validate:"required,hostname|ip"`
	Port                 int    `yaml:"port" env:"POSTGRES_PORT" validate:"required,min=1,max=65535"`
	MinConns             int    `yaml:"min-conns" env:"POSTGRES_MIN_CONNS" validate:"required,min=0"`
	MaxConns             int    `yaml:"max-conns" env:"POSTGRES_MAX_CONNS" validate:"required,min=1"`
	MigPath              string `yaml:"mig-path" env:"POSTGRES_MIG_PATH" validate:"required"`
	QueryTimeoutSeconds  int    `yaml:"query-timeout-seconds" env:"DB_QUERY_TIMEOUT_SECONDS" env-default:"10" validate:"min=1,max=120"`
	SlowQueryThresholdMS int    `yaml:"slow-query-threshold-ms" env:"DB_SLOW_QUERY_THRESHOLD_MS" env-default:"250" validate:"min=1,max=60000"`
}
