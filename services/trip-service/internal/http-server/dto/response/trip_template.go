package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type TripTemplateAccess struct {
	Role         string `json:"role"`
	Source       string `json:"source"`
	CanUse       bool   `json:"canUse"`
	CanEdit      bool   `json:"canEdit"`
	CanArchive   bool   `json:"canArchive"`
	CanDuplicate bool   `json:"canDuplicate"`
}

type TripTemplate struct {
	ID                     string                        `json:"id"`
	WorkspaceID            *string                       `json:"workspaceId"`
	CreatedByUserID        string                        `json:"createdByUserId"`
	SourceTripID           *string                       `json:"sourceTripId"`
	Title                  string                        `json:"title"`
	Description            *string                       `json:"description"`
	DestinationHint        *string                       `json:"destinationHint"`
	DurationDays           int32                         `json:"durationDays"`
	DefaultCurrency        *string                       `json:"defaultCurrency"`
	Visibility             entity.TripTemplateVisibility `json:"visibility"`
	Tags                   []string                      `json:"tags"`
	EstimatedTotalAmount   *float64                      `json:"estimatedTotalAmount"`
	EstimatedTotalCurrency *string                       `json:"estimatedTotalCurrency"`
	Status                 entity.TripTemplateStatus     `json:"status"`
	Access                 TripTemplateAccess            `json:"access"`
	CreatedAt              time.Time                     `json:"createdAt"`
	UpdatedAt              time.Time                     `json:"updatedAt"`
	ArchivedAt             *time.Time                    `json:"archivedAt,omitempty"`
}

type TripTemplateDetail struct {
	TripTemplate
	TemplateJSON any `json:"templateJson"`
}

type ListTripTemplates struct {
	Templates  []TripTemplate `json:"templates"`
	Items      []TripTemplate `json:"items"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	NextCursor *string        `json:"nextCursor"`
}

func NewListTripTemplates(items []appdto.TripTemplateWithAccess, limit, offset int) ListTripTemplates {
	templates := make([]TripTemplate, 0, len(items))
	for _, item := range items {
		templates = append(templates, NewTripTemplate(item))
	}
	return ListTripTemplates{
		Templates: templates,
		Items:     templates,
		Limit:     limit,
		Offset:    offset,
	}
}

func NewTripTemplate(item appdto.TripTemplateWithAccess) TripTemplate {
	template := item.Template
	return TripTemplate{
		ID:                     template.ID.String(),
		WorkspaceID:            uuidStringPtr(template.WorkspaceID),
		CreatedByUserID:        template.CreatedByUserID.String(),
		SourceTripID:           uuidStringPtr(template.SourceTripID),
		Title:                  template.Title,
		Description:            template.Description,
		DestinationHint:        template.DestinationHint,
		DurationDays:           template.DurationDays,
		DefaultCurrency:        template.DefaultCurrency,
		Visibility:             template.Visibility,
		Tags:                   template.Tags,
		EstimatedTotalAmount:   template.EstimatedTotalAmount,
		EstimatedTotalCurrency: template.EstimatedTotalCurrency,
		Status:                 template.Status,
		Access:                 NewTripTemplateAccess(item.Access),
		CreatedAt:              template.CreatedAt,
		UpdatedAt:              template.UpdatedAt,
		ArchivedAt:             template.ArchivedAt,
	}
}

func NewTripTemplateDetail(item appdto.TripTemplateWithAccess) TripTemplateDetail {
	var templateJSON any
	if len(item.Template.TemplateJSON) > 0 {
		if err := json.Unmarshal(item.Template.TemplateJSON, &templateJSON); err != nil {
			templateJSON = map[string]any{}
		}
	}
	return TripTemplateDetail{
		TripTemplate: NewTripTemplate(item),
		TemplateJSON: templateJSON,
	}
}

func NewTripTemplateAccess(access appdto.TripTemplateAccess) TripTemplateAccess {
	return TripTemplateAccess{
		Role:         access.Role,
		Source:       access.Source,
		CanUse:       access.CanUse,
		CanEdit:      access.CanEdit,
		CanArchive:   access.CanArchive,
		CanDuplicate: access.CanDuplicate,
	}
}

func uuidStringPtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}
