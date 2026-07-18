package copilot

import (
	"fmt"
	"strings"
)

func fallbackResponse(intent Intent, context SafeContext, available []Action, language string) AIResponse {
	answer := fallbackText(language, "general")
	sources := []string{"command_center"}
	switch intent {
	case IntentUnsafeMutationRequest:
		answer = fallbackText(language, "unsafe")
		sources = []string{"app_help"}
	case IntentOutOfScope:
		answer = fallbackText(language, "scope")
		sources = []string{"app_help"}
	case IntentExplainHealth:
		if health := context.Health; health != nil {
			answer = fmt.Sprintf(fallbackText(language, "health"), health["score"], health["level"], stringValue(health["summary"]))
		} else {
			answer = fallbackText(language, "health_missing")
		}
		sources = []string{"trip_health", "command_center"}
	case IntentExplainVerification:
		if readiness := context.Verification; readiness != nil {
			answer = fmt.Sprintf(fallbackText(language, "verification"), readiness["score"], readiness["level"], readiness["topIssueCount"])
		} else {
			answer = fallbackText(language, "verification_missing")
		}
		sources = []string{"real_world_verification"}
	case IntentExplainBudget:
		if budget := context.Budget; budget != nil {
			answer = fmt.Sprintf(fallbackText(language, "budget"), budget["level"], budget["riskLevel"], stringValue(budget["summary"]))
		} else {
			answer = fallbackText(language, "budget_missing")
		}
		sources = []string{"budget_confidence"}
	case IntentExplainRoute:
		if route := context.Route; route != nil {
			answer = fmt.Sprintf(fallbackText(language, "route"), route["legCount"], route["missingTransportCount"])
		} else {
			answer = fallbackText(language, "route_missing")
		}
		sources = []string{"route_summary"}
	case IntentExplainGroupReadiness:
		if group := context.Group; group != nil {
			answer = fmt.Sprintf(fallbackText(language, "group"), group["level"], group["membersNeedingAttention"])
		} else {
			answer = fallbackText(language, "group_missing")
		}
		sources = []string{"group_readiness"}
	case IntentExplainChecklist:
		if checklist := context.Checklist; checklist != nil {
			answer = fmt.Sprintf(fallbackText(language, "checklist"), checklist["completedCount"], checklist["totalCount"], checklist["overdueCount"])
		} else {
			answer = fallbackText(language, "checklist_missing")
		}
		sources = []string{"checklist_summary", "reminders_summary"}
	case IntentExplainExpenses:
		answer = fallbackText(language, "expenses")
		sources = []string{"expenses_summary"}
	case IntentExplainApproval:
		if approval := context.Approval; approval != nil {
			answer = fmt.Sprintf(fallbackText(language, "approval"), approval["status"])
		} else {
			answer = fallbackText(language, "approval_missing")
		}
		sources = []string{"approval_status", "policy_evaluation"}
	case IntentHowTo, IntentExplainFeature, IntentFindSection:
		answer = fallbackText(language, "how_to")
		sources = []string{"app_help"}
	default:
		if health := context.Health; health != nil {
			answer = fmt.Sprintf(fallbackText(language, "next"), stringValue(health["summary"]))
			sources = []string{"trip_health", "command_center"}
		}
	}
	if len(context.Unavailable) > 0 && !strings.Contains(answer, "could not") {
		answer += " " + fallbackText(language, "partial")
	}
	return AIResponse{
		Answer:      answer,
		Actions:     preferredActions(intent, available),
		SourceTypes: sources,
		Warnings:    unavailableWarnings(context.Unavailable, language),
	}
}

func unavailableWarnings(unavailable []string, language string) []string {
	if len(unavailable) == 0 {
		return []string{}
	}
	return []string{fallbackText(language, "partial")}
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func fallbackText(language, key string) string {
	messages := map[string]map[string]string{
		"en": {
			"unsafe": "I can’t make changes, bookings, payments, or send messages for you. I can point you to the relevant screen so you can review the action yourself.", "scope": "I can’t verify legal, medical, safety, or live travel conditions here. Please confirm those details with official sources.", "health": "Trip Health is %v (%v). %s", "health_missing": "I could not check Trip Health right now. You can still review the health panel directly.", "verification": "Real-world readiness is %v (%v), with %v item(s) needing review. This is not a booking or price guarantee.", "verification_missing": "I could not check real-world readiness right now. Review the verification panel before relying on key travel details.", "budget": "Budget Confidence is %v (%v risk). %s", "budget_missing": "I could not check Budget Confidence right now. Review the budget panel for the latest estimates.", "route": "Your route has %v leg(s), with %v leg(s) missing selected transport.", "route_missing": "I could not load the route summary right now.", "group": "Group Readiness is %v. %v member(s) need attention; I only use shared readiness summaries, not private notes.", "group_missing": "I could not check group readiness right now.", "checklist": "Your checklist has %v of %v item(s) complete, with %v overdue item(s).", "checklist_missing": "Open the checklist to review packing and preparation items.", "expenses": "Open Expenses to review totals, settlements, and receipt upload. Copilot does not read raw receipt OCR or private expense notes.", "approval": "Approval status is %v. Review the approval and policy panels for current blockers or warnings.", "approval_missing": "I could not check approval status right now.", "how_to": "I can guide you to the relevant trip section. Copilot only suggests links; it cannot change trip data.", "next": "Start with Trip Health: %s", "general": "I can help you review this trip using its current summaries.", "partial": "Some trip summaries are temporarily unavailable.", "permission_note": "You can view this trip, but an editor must make trip changes.",
		},
		"es": {
			"unsafe": "No puedo hacer cambios, reservas, pagos ni enviar mensajes. Puedo llevarte a la pantalla adecuada para revisar la acción.", "scope": "No puedo verificar condiciones legales, médicas, de seguridad o en tiempo real. Confírmalas con fuentes oficiales.", "health": "La salud del viaje es %v (%v). %s", "health_missing": "No pude comprobar la salud del viaje ahora mismo.", "verification": "La preparación para el mundo real es %v (%v), con %v elemento(s) que necesitan revisión. No es una garantía de reserva ni de precio.", "verification_missing": "No pude comprobar la preparación para el mundo real. Revisa la verificación antes de depender de detalles importantes.", "budget": "La confianza del presupuesto es %v (riesgo %v). %s", "budget_missing": "No pude comprobar la confianza del presupuesto ahora mismo.", "route": "Tu ruta tiene %v tramo(s), con %v sin transporte seleccionado.", "route_missing": "No pude cargar el resumen de la ruta ahora mismo.", "group": "La preparación del grupo es %v. %v persona(s) necesitan atención; solo uso resúmenes compartidos.", "group_missing": "No pude comprobar la preparación del grupo ahora mismo.", "checklist": "Tu lista tiene %v de %v elementos completados y %v vencidos.", "checklist_missing": "Abre la lista para revisar los preparativos.", "expenses": "Abre Gastos para revisar totales, liquidaciones y recibos. Copilot no lee OCR de recibos ni notas privadas.", "approval": "El estado de aprobación es %v. Revisa aprobación y políticas para ver bloqueos o avisos.", "approval_missing": "No pude comprobar la aprobación ahora mismo.", "how_to": "Puedo guiarte a la sección adecuada. Copilot solo sugiere enlaces y no puede cambiar datos.", "next": "Empieza con la salud del viaje: %s", "general": "Puedo ayudarte a revisar los resúmenes actuales del viaje.", "partial": "Algunos resúmenes del viaje no están disponibles temporalmente.", "permission_note": "Puedes ver este viaje, pero un editor debe realizar los cambios.",
		},
		"fr": {
			"unsafe": "Je ne peux pas effectuer de modifications, réservations, paiements ou envois. Je peux vous diriger vers l’écran approprié.", "scope": "Je ne peux pas vérifier les conditions juridiques, médicales, de sécurité ou en direct. Consultez des sources officielles.", "health": "La santé du voyage est de %v (%v). %s", "health_missing": "Je ne peux pas vérifier la santé du voyage pour le moment.", "verification": "La préparation aux conditions réelles est de %v (%v), avec %v élément(s) à examiner. Ceci ne garantit ni réservation ni prix.", "verification_missing": "Je ne peux pas vérifier la préparation aux conditions réelles pour le moment. Consultez la vérification avant de vous fier aux détails essentiels.", "budget": "La fiabilité du budget est %v (risque %v). %s", "budget_missing": "Je ne peux pas vérifier la fiabilité du budget pour le moment.", "route": "Votre itinéraire compte %v étape(s), dont %v sans transport sélectionné.", "route_missing": "Je ne peux pas charger le résumé de l’itinéraire pour le moment.", "group": "La préparation du groupe est %v. %v membre(s) demandent de l’attention; j’utilise uniquement des résumés partagés.", "group_missing": "Je ne peux pas vérifier la préparation du groupe pour le moment.", "checklist": "Votre liste compte %v éléments terminés sur %v, dont %v en retard.", "checklist_missing": "Ouvrez la liste pour examiner les préparatifs.", "expenses": "Ouvrez Dépenses pour examiner les totaux, règlements et reçus. Copilot ne lit pas l’OCR brut ni les notes privées.", "approval": "Le statut d’approbation est %v. Consultez l’approbation et les politiques pour les blocages ou avertissements.", "approval_missing": "Je ne peux pas vérifier l’approbation pour le moment.", "how_to": "Je peux vous guider vers la bonne section. Copilot suggère seulement des liens et ne peut pas modifier les données.", "next": "Commencez par la santé du voyage : %s", "general": "Je peux vous aider à examiner les résumés actuels du voyage.", "partial": "Certains résumés du voyage sont temporairement indisponibles.", "permission_note": "Vous pouvez consulter ce voyage, mais un éditeur doit effectuer les changements.",
		},
		"uk": {
			"unsafe": "Я не можу вносити зміни, бронювати, оплачувати чи надсилати повідомлення. Я можу спрямувати вас до потрібного екрана.", "scope": "Я не можу перевірити юридичні, медичні, безпекові чи поточні умови. Будь ласка, перевірте офіційні джерела.", "health": "Стан подорожі: %v (%v). %s", "health_missing": "Зараз не вдалося перевірити стан подорожі.", "verification": "Готовність до реальних умов: %v (%v), %v пункт(ів) потребують перевірки. Це не є гарантією бронювання чи ціни.", "verification_missing": "Зараз не вдалося перевірити готовність до реальних умов. Перегляньте перевірку перед тим, як покладатися на ключові деталі.", "budget": "Надійність бюджету: %v (ризик %v). %s", "budget_missing": "Зараз не вдалося перевірити надійність бюджету.", "route": "У маршруті %v етап(ів), і для %v не вибрано транспорт.", "route_missing": "Зараз не вдалося завантажити підсумок маршруту.", "group": "Готовність групи: %v. Уваги потребують %v учасник(и); я використовую лише спільні зведення.", "group_missing": "Зараз не вдалося перевірити готовність групи.", "checklist": "У списку виконано %v з %v; прострочено %v.", "checklist_missing": "Відкрийте список, щоб переглянути підготовку.", "expenses": "Відкрийте витрати, щоб переглянути підсумки, розрахунки та чеки. Copilot не читає необроблений OCR чи приватні нотатки.", "approval": "Статус схвалення: %v. Перегляньте схвалення й політики для блокерів або попереджень.", "approval_missing": "Зараз не вдалося перевірити схвалення.", "how_to": "Я можу спрямувати вас до потрібного розділу. Copilot лише пропонує посилання та не може змінювати дані.", "next": "Почніть зі стану подорожі: %s", "general": "Я можу допомогти переглянути поточні зведення подорожі.", "partial": "Деякі зведення подорожі тимчасово недоступні.", "permission_note": "Ви можете переглядати цю подорож, але зміни має зробити редактор.",
		},
	}
	if value, ok := messages[language][key]; ok {
		return value
	}
	return messages["en"][key]
}
