package email

// This file is intentionally small. The package is constructed via NewSender
// (provider selection) in the composition root. It stays in pkg because it owns
// reusable email message and delivery plumbing, while notification-specific
// rendering and policy live under internal/emailnotifications.
