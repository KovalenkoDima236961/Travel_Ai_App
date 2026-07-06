package request

import appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"

// SubmitApproval is the POST /trips/{id}/approval/submit body. Both fields are
// optional; the checklist (not the request) decides whether submission proceeds.
type SubmitApproval struct {
	Note                 string   `json:"note"`
	AcknowledgedWarnings []string `json:"acknowledgedWarnings"`
}

func (r SubmitApproval) ToInput() appdto.SubmitApprovalInput {
	return appdto.SubmitApprovalInput{
		Note:                 r.Note,
		AcknowledgedWarnings: r.AcknowledgedWarnings,
	}
}

// ApprovalDecision is the body for approve and request-changes. decisionNote is
// optional for approve and required (1-1000 chars) for request-changes; the
// service enforces the per-action rule.
type ApprovalDecision struct {
	DecisionNote string `json:"decisionNote"`
}

func (r ApprovalDecision) ToInput() appdto.ApprovalDecisionInput {
	return appdto.ApprovalDecisionInput{DecisionNote: r.DecisionNote}
}

// CancelApproval is the POST /trips/{id}/approval/cancel body.
type CancelApproval struct {
	Note string `json:"note"`
}

func (r CancelApproval) ToInput() appdto.CancelApprovalInput {
	return appdto.CancelApprovalInput{Note: r.Note}
}
