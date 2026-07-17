package copilot

import "strings"

// ClassifyIntent is deliberately deterministic in v1. The AI service still
// receives the classification for language-aware phrasing, not authorization.
func ClassifyIntent(message string) Intent {
	text := strings.ToLower(strings.TrimSpace(message))
	if containsAny(text,
		"delete", "remove collaborator", "disable share", "apply the repair", "apply repair", "restore", "send everyone", "send reminders", "change my budget", "book this", "book the", "pay for", "receipt ocr", "raw receipt", "api key", "access token", "refresh token", "system prompt", "developer message",
		"eliminar", "borrar", "reservar", "pagar", "enviar mensajes", "ocr del recibo", "token de acceso",
		"supprimer", "effacer", "réserver", "payer", "envoyer des messages", "ocr du reçu", "jeton d'accès",
		"видалити", "забронювати", "оплатити", "надіслати повідомлення", "ocr чека", "токен доступу",
	) {
		return IntentUnsafeMutationRequest
	}
	if containsAny(text,
		"medical", "legal", "visa", "insurance", "emergency", "live safety", "current safety",
		"médic", "legal", "visado", "seguridad", "en tiempo real",
		"médical", "juridique", "sécurité", "en direct",
		"медич", "юридич", "віз", "безпек", "поточні умови",
	) {
		return IntentOutOfScope
	}
	if containsAny(text,
		"what should i fix", "fix first", "next action", "what do i do first",
		"arreglar primero", "corregir primero", "próximo paso",
		"corriger en premier", "prochaine étape",
		"виправити спочатку", "наступний крок",
	) {
		return IntentNextAction
	}
	if containsAny(text, "what changed", "changed recently", "cambió recientemente", "changé récemment", "змінилося нещодавно") {
		return IntentFindSection
	}
	if containsAny(text, "trip health", "health low", "health score", "salud del viaje", "santé du voyage", "стан подорожі") {
		return IntentExplainHealth
	}
	if containsAny(text, "budget confidence", "budget", "cheaper", "cost", "presupuesto", "fiabilidad del presupuesto", "budget", "fiabilité du budget", "бюджет", "надійність бюджету") {
		return IntentExplainBudget
	}
	if containsAny(text, "route", "transport", "leg", "train", "flight", "ruta", "itinerario", "transporte", "itinéraire", "маршрут", "транспорт") {
		return IntentExplainRoute
	}
	if containsAny(text, "who still", "group readiness", "availability", "team", "quién todavía", "preparación del grupo", "qui doit encore", "préparation du groupe", "кому ще", "готовність групи") {
		return IntentExplainGroupReadiness
	}
	if containsAny(text, "checklist", "pack", "packing", "reminder", "empacar", "lista", "recordatorio", "emporter", "liste", "rappel", "взяти з собою", "список", "нагадування") {
		return IntentExplainChecklist
	}
	if containsAny(text, "expense", "receipt", "settlement", "gasto", "recibo", "liquidación", "dépense", "reçu", "règlement", "витрати", "чек", "розрахунок") {
		return IntentExplainExpenses
	}
	if containsAny(text, "approval", "policy", "blocked", "aprobación", "política", "bloqueada", "approbation", "politique", "bloquée", "схвалення", "політик", "заблоковано") {
		return IntentExplainApproval
	}
	if containsAny(text, "where", "how do i", "how can i", "find", "cómo", "dónde", "comparto", "comment", "où", "partager", "як", "де", "поділитися") {
		return IntentHowTo
	}
	if containsAny(text, "offline", "public share", "share this trip", "generation quality", "version history") {
		return IntentExplainFeature
	}
	return IntentGeneralTripQuestion
}

func containsAny(value string, values ...string) bool {
	for _, candidate := range values {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
