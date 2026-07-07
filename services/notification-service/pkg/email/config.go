package email

// Config selects and configures the email sender. It is built from the
// service's loaded configuration (internal/config) in the composition root, so
// this package stays free of any config-file/env concerns.
type Config struct {
	// Provider selects the implementation: "mock" or "smtp".
	Provider string
	SMTP     SMTPConfig
}

// SMTPConfig holds the settings the SMTP sender needs. Password is read from the
// environment only and must never be logged.
type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool
}

// Provider names accepted by NewSender.
const (
	ProviderMock = "mock"
	ProviderSMTP = "smtp"
)
