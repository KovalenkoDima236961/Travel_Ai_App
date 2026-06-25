package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/email"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/emailnotifications"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/handler"
	notificationrepo "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/infrastructure/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/users"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/storage/postgres"
)

// container holds the wired dependencies. It is a small, explicit composition
// root — no DI framework — assembled in buildContainer to match the existing
// Go services' style.
type container struct {
	db     *postgres.DB
	router http.Handler
}

// buildContainer constructs and wires dependencies in order:
// postgres (with auto-migrations) -> repository -> service -> handlers ->
// router. Long-lived resources register themselves with the closer.
func buildContainer(ctx context.Context, cfg *config.Config, log *zap.Logger) (*container, error) {
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}
	closer.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})

	repo := notificationrepo.New(db)
	svc := notifications.New(repo, log)
	preferenceSvc := preferences.New(repo, log)

	// Email fan-out: select the sender (mock/smtp), build the recipient lookup
	// client (Auth Service owns email in v1), and wire the orchestration. Sender
	// and client construction are startup-validated, so a misconfiguration fails
	// fast here rather than silently dropping mail.
	emailSender, err := email.NewSender(email.Config{
		Provider: cfg.Email.Provider,
		SMTP: email.SMTPConfig{
			Host:      cfg.Email.SMTP.Host,
			Port:      cfg.Email.SMTP.Port,
			Username:  cfg.Email.SMTP.Username,
			Password:  cfg.Email.SMTP.Password,
			FromEmail: cfg.Email.SMTP.FromEmail,
			FromName:  cfg.Email.SMTP.FromName,
			UseTLS:    cfg.Email.SMTP.UseTLS,
		},
	}, log)
	if err != nil {
		return nil, fmt.Errorf("init email sender: %w", err)
	}

	userLookup, err := users.New(users.Config{
		BaseURL:        cfg.Users.AuthServiceURL,
		Token:          cfg.Internal.ServiceToken,
		TimeoutSeconds: cfg.Users.TimeoutSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("init user lookup client: %w", err)
	}

	emailSvc := emailnotifications.New(emailnotifications.Config{
		Enabled:          cfg.Email.Enabled,
		FailOpen:         cfg.Email.FailOpen,
		PublicWebBaseURL: cfg.Email.PublicWebBaseURL,
		Types:            cfg.EmailNotificationTypes(),
	}, userLookup, emailSender, log)

	notificationHandler := handler.New(svc, log, preferenceSvc)
	internalHandler := handler.NewInternal(svc, emailSvc, log, preferenceSvc)
	readinessHandler := httpserver.NewReadinessHandler(db, log)

	router := httpserver.NewRouter(
		log,
		notificationHandler,
		internalHandler,
		readinessHandler,
		cfg.CORS,
		cfg.JWT,
		cfg.Internal,
	)

	return &container{
		db:     db,
		router: router,
	}, nil
}
