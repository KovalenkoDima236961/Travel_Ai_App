package security

import "github.com/prometheus/client_golang/prometheus"

var (
	ShareUnlockAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "share_unlock_attempts_total", Help: "Public share unlock attempts.",
	}, []string{"outcome"})
	ReceiptUploadRejected = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "receipt_upload_rejected_total", Help: "Receipt uploads rejected by security checks.",
	}, []string{"reason"})
	ReceiptDownloadDenied = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "receipt_download_denied_total", Help: "Receipt file downloads denied.",
	}, []string{"reason"})
	PermissionDenied = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "permission_denied_total", Help: "Trip permission checks denied.",
	}, []string{"resource"})
	SecurityAuditEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "security_audit_events_total", Help: "Safe structured security audit events emitted by Trip Service.",
	}, []string{"action", "outcome"})
)

func init() {
	prometheus.MustRegister(ShareUnlockAttempts, ReceiptUploadRejected, ReceiptDownloadDenied, PermissionDenied, SecurityAuditEvents)
}
