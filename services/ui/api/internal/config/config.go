package config

import (
	"context"
	"time"

	"github.com/sethvargo/go-envconfig"
)

// Config holds runtime configuration for the UI API service.
type Config struct {
	Addr                    string        `env:"ADDR,default=:8080"`
	DBDSN                   string        `env:"DB_DSN,required"`
	JWTSigningKey           string        `env:"JWT_SIGNING_KEY,required"`
	JWTRefreshKey           string        `env:"JWT_REFRESH_KEY,required"`
	SMTPHost                string        `env:"SMTP_HOST"`
	SMTPPort                int           `env:"SMTP_PORT,default=587"`
	SMTPUser                string        `env:"SMTP_USER"`
	SMTPPassword            string        `env:"SMTP_PASS"`
	OIDCIssuer              string        `env:"OIDC_ISSUER"`
	OIDCClientID            string        `env:"OIDC_CLIENT_ID"`
	OIDCClientSecret        string        `env:"OIDC_CLIENT_SECRET"`
	APIBaseURL              string        `env:"API_BASE_URL,required"`
	OTLPEndpoint            string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	AllowedOrigins          []string      `env:"CORS_ALLOWED_ORIGINS,default=http://localhost:5173"`
	AccessTokenTTL          time.Duration `env:"ACCESS_TOKEN_TTL,default=15m"`
	RefreshTokenTTL         time.Duration `env:"REFRESH_TOKEN_TTL,default=336h"`
	RefreshTokenGracePeriod time.Duration `env:"REFRESH_TOKEN_GRACE_PERIOD,default=5m"`
	CookieDomain            string        `env:"COOKIE_DOMAIN"`
	CookieSecure            bool          `env:"COOKIE_SECURE,default=false"`
}

// Load returns a Config populated from environment variables.
func Load(ctx context.Context) (Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
