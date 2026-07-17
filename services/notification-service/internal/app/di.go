package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/controls"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/digests"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/emailnotifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/httpserver"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/httpserver/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
	pushsvc "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/push"
	notificationrepo "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/stream"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/users"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/email"
	pushdelivery "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/push"
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
// router. Long-lived resources register themselves with shutdown.
func buildContainer(
	ctx context.Context,
	cfg *config.Config,
	log *zap.Logger,
	shutdown *closer.Stack,
) (*container, error) {
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}
	shutdown.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})

	repo := notificationrepo.New(db)
	svc := notifications.New(repo, log).
		WithDedupeWindow(cfg.NotificationDedupeWindow()).
		WithGroupingWindow(cfg.NotificationGroupingWindow())
	preferenceSvc := preferences.New(repo, log)
	controlsSvc := controls.New(repo, log)
	streamCfg := stream.Normalize(stream.Config{
		Enabled:               cfg.SSE.Enabled,
		HeartbeatInterval:     cfg.SSEHeartbeatInterval(),
		WriteTimeout:          cfg.SSEWriteTimeout(),
		MaxConnectionsPerUser: cfg.SSE.MaxConnectionsPerUser,
	})
	streamManager := stream.NewManager(streamCfg, log)

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

	pushCfg := pushsvc.Config{
		Enabled:         cfg.WebPush.Enabled,
		VAPIDPublicKey:  cfg.WebPush.VAPIDPublicKey,
		VAPIDPrivateKey: cfg.WebPush.VAPIDPrivateKey,
		FailOpen:        cfg.WebPush.FailOpen,
	}
	pushSender, err := pushdelivery.NewSender(pushdelivery.Config{
		Enabled:         cfg.WebPush.Enabled,
		VAPIDPublicKey:  cfg.WebPush.VAPIDPublicKey,
		VAPIDPrivateKey: cfg.WebPush.VAPIDPrivateKey,
		Subject:         cfg.WebPush.Subject,
		Timeout:         cfg.WebPushTimeout(),
		TTLSeconds:      cfg.WebPush.TTLSeconds,
		Urgency:         cfg.WebPush.Urgency,
	}, log)
	if err != nil {
		return nil, fmt.Errorf("init web push sender: %w", err)
	}
	pushSvc := pushsvc.New(pushCfg, repo, pushSender, log)
	digestSvc := digests.New(repo, userLookup, emailSender, pushSvc, digests.Config{
		PublicWebBaseURL: cfg.Email.PublicWebBaseURL,
		MaxAttempts:      cfg.Digest.MaxAttempts,
		RetryDelay:       cfg.NotificationDigestRetryDelay(),
	}, log)

	notificationHandler := handler.New(svc, log, preferenceSvc).
		EnableStream(streamManager, streamCfg).
		EnablePush(pushSvc).
		EnableControls(controlsSvc).
		EnableDigests(digestSvc)
	internalHandler := handler.NewInternal(svc, emailSvc, log, preferenceSvc).
		EnableStream(streamManager).
		EnablePush(pushSvc).
		EnableNoiseControl(controlsSvc, digestSvc)
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
