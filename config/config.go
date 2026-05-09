package config

import "time"

type Config struct {
	App      AppConfig
	DB       DBConfig
	Kafka    KafkaConfig
	Midtrans MidtransConfig
	Redis    RedisConfig
	OTEL     OTELConfig
}

type AppConfig struct {
	Name        string `env:"APP_NAME" envDefault:"payment-service"`
	Env         string `env:"APP_ENV" envDefault:"development"`
	Port        int    `env:"APP_PORT" envDefault:"8080"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"DEBUG"`
	LogFilePath string `env:"LOG_FILE_PATH" envDefault:""`
	Version     string `env:"APP_VERSION" envDefault:"v1"`
}

type DBConfig struct {
	Host            string        `env:"DB_HOST" envDefault:"localhost"`
	Port            int           `env:"DB_PORT" envDefault:"5432"`
	Name            string        `env:"DB_NAME,required"`
	User            string        `env:"DB_USER,required"`
	Password        string        `env:"DB_PASSWORD,required"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" envDefault:"10"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"5m"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE" envDefault:"1m"`
}

type KafkaConfig struct {
	Brokers []string `env:"KAFKA_BROKERS" envSeparator:"," envDefault:"localhost:9092"`
	Topic   string   `env:"KAFKA_TOPIC" envDefault:"payment.payment.settled"`
}

type MidtransConfig struct {
	ServerKey     string        `env:"MIDTRANS_SERVER_KEY,required"`
	ClientKey     string        `env:"MIDTRANS_CLIENT_KEY"`
	IsProd        bool          `env:"MIDTRANS_IS_PROD" envDefault:"false"`
	Timeout       time.Duration `env:"MIDTRANS_TIMEOUT" envDefault:"10s"`
	CBMaxRequests uint32        `env:"MIDTRANS_CB_MAX_REQUESTS" envDefault:"3"`
	CBInterval    time.Duration `env:"MIDTRANS_CB_INTERVAL" envDefault:"1m"`
	CBTimeout     time.Duration `env:"MIDTRANS_CB_TIMEOUT" envDefault:"30s"`
}

type RedisConfig struct {
	Host     string `env:"REDIS_HOST" envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT" envDefault:"6379"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

type OTELConfig struct {
	Enabled        bool    `env:"OTEL_ENABLED" envDefault:"false"`
	Endpoint       string  `env:"OTEL_ENDPOINT" envDefault:"localhost:4317"`
	ServiceName    string  `env:"OTEL_SERVICE_NAME" envDefault:"payment-service"`
	MetricsEnabled bool    `env:"OTEL_METRICS_ENABLED" envDefault:"true"`
	TracingEnabled bool    `env:"OTEL_TRACING_ENABLED" envDefault:"true"`
	SampleRate     float64 `env:"OTEL_SAMPLE_RATE" envDefault:"1.0"`
}
