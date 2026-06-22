package redis

import "time"

// Config describes how to connect to Redis. Tags mirror the project's
// yaml + env (cleanenv) convention so it can be embedded in the app config.
type Config struct {
	Host            string        `yaml:"host" env:"REDIS_HOST" env-default:"localhost" validate:"required,hostname|ip"`
	Port            int           `yaml:"port" env:"REDIS_PORT" env-default:"6379" validate:"required,min=1,max=65535"`
	Username        string        `yaml:"username" env:"REDIS_USERNAME"`
	Password        string        `yaml:"password" env:"REDIS_PASSWORD"`
	Database        int           `yaml:"database" env:"REDIS_DATABASE" env-default:"0" validate:"min=0"`
	Protocol        int           `yaml:"protocol" env:"REDIS_PROTOCOL" env-default:"3" validate:"oneof=2 3"`
	Retries         int           `yaml:"retries" env:"REDIS_RETRIES" env-default:"3" validate:"min=0"`
	MinRetryBackoff time.Duration `yaml:"min_retry_backoff" env:"REDIS_MIN_RETRY_BACKOFF" env-default:"8ms"`
	MaxRetryBackoff time.Duration `yaml:"max_retry_backoff" env:"REDIS_MAX_RETRY_BACKOFF" env-default:"512ms"`
	DialTimeout     time.Duration `yaml:"dial_timeout" env:"REDIS_DIAL_TIMEOUT" env-default:"5s"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"REDIS_READ_TIMEOUT" env-default:"3s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"REDIS_WRITE_TIMEOUT" env-default:"3s"`
	PoolSize        int           `yaml:"pool_size" env:"REDIS_POOL_SIZE" env-default:"10" validate:"min=1"`
	MinIdleConns    int           `yaml:"min_idle_conns" env:"REDIS_MIN_IDLE_CONNS" env-default:"0" validate:"min=0"`
}
