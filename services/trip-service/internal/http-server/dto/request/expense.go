package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type ExpenseAmount struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type CreateTripExpense struct {
	Title               string                     `json:"title"`
	Description         *string                    `json:"description"`
	Amount              ExpenseAmount              `json:"amount"`
	Category            entity.ExpenseCategory     `json:"category"`
	ExpenseDate         string                     `json:"expenseDate"`
	PaidByUserID        string                     `json:"paidByUserId"`
	SplitType           entity.ExpenseSplitType    `json:"splitType"`
	ParticipantUserIDs  []string                   `json:"participantUserIds"`
	CustomShares        []ExpenseCustomAmount      `json:"customShares"`
	CustomPercentages   []ExpenseCustomPercentage  `json:"customPercentages"`
	LinkedItinerary     *appdto.LinkedItineraryRef `json:"linkedItinerary"`
	LinkedRouteLegID    *string                    `json:"linkedRouteLegId"`
	LinkedAccommodation bool                       `json:"linkedAccommodation"`
	Notes               *string                    `json:"notes"`
	Metadata            map[string]any             `json:"metadata"`
}

type ExpenseCustomAmount struct {
	UserID   string  `json:"userId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type ExpenseCustomPercentage struct {
	UserID     string  `json:"userId"`
	Percentage float64 `json:"percentage"`
}

func (r CreateTripExpense) ToInput() (appdto.CreateExpenseInput, error) {
	paidByUserID, err := parseExpenseUUID(r.PaidByUserID, "paidByUserId")
	if err != nil {
		return appdto.CreateExpenseInput{}, err
	}
	expenseDate, err := parseExpenseDate(r.ExpenseDate, "expenseDate")
	if err != nil {
		return appdto.CreateExpenseInput{}, err
	}
	participantIDs, err := parseExpenseUUIDs(r.ParticipantUserIDs, "participantUserIds")
	if err != nil {
		return appdto.CreateExpenseInput{}, err
	}
	customShares, err := r.customShareInputs()
	if err != nil {
		return appdto.CreateExpenseInput{}, err
	}
	customPercentages, err := r.customPercentageInputs()
	if err != nil {
		return appdto.CreateExpenseInput{}, err
	}
	return appdto.CreateExpenseInput{
		Title:               r.Title,
		Description:         r.Description,
		Amount:              appdto.MoneyAmount{Amount: r.Amount.Amount, Currency: r.Amount.Currency},
		Category:            r.Category,
		ExpenseDate:         expenseDate,
		PaidByUserID:        paidByUserID,
		SplitType:           r.SplitType,
		ParticipantUserIDs:  participantIDs,
		CustomShares:        customShares,
		CustomPercentages:   customPercentages,
		LinkedItinerary:     r.LinkedItinerary,
		LinkedRouteLegID:    r.LinkedRouteLegID,
		LinkedAccommodation: r.LinkedAccommodation,
		Notes:               r.Notes,
		Metadata:            r.Metadata,
	}, nil
}

func (r CreateTripExpense) customShareInputs() ([]appdto.ExpenseCustomAmount, error) {
	out := make([]appdto.ExpenseCustomAmount, 0, len(r.CustomShares))
	for _, share := range r.CustomShares {
		userID, err := parseExpenseUUID(share.UserID, "customShares.userId")
		if err != nil {
			return nil, err
		}
		out = append(out, appdto.ExpenseCustomAmount{
			UserID:   userID,
			Amount:   share.Amount,
			Currency: share.Currency,
		})
	}
	return out, nil
}

func (r CreateTripExpense) customPercentageInputs() ([]appdto.ExpenseCustomPercentage, error) {
	out := make([]appdto.ExpenseCustomPercentage, 0, len(r.CustomPercentages))
	for _, item := range r.CustomPercentages {
		userID, err := parseExpenseUUID(item.UserID, "customPercentages.userId")
		if err != nil {
			return nil, err
		}
		out = append(out, appdto.ExpenseCustomPercentage{
			UserID:     userID,
			Percentage: item.Percentage,
		})
	}
	return out, nil
}

type UpdateTripExpense struct {
	raw map[string]json.RawMessage
}

func (r *UpdateTripExpense) UnmarshalJSON(data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 {
		r.raw = map[string]json.RawMessage{}
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.raw = raw
	return nil
}

func (r UpdateTripExpense) ToInput() (appdto.UpdateExpenseInput, error) {
	var in appdto.UpdateExpenseInput
	if raw, ok := r.raw["title"]; ok && !isJSONNull(raw) {
		value, err := decodeExpenseString(raw, "title")
		if err != nil {
			return in, err
		}
		in.Title = &value
	}
	if raw, ok := r.raw["description"]; ok {
		if isJSONNull(raw) {
			in.ClearDescription = true
		} else {
			value, err := decodeExpenseString(raw, "description")
			if err != nil {
				return in, err
			}
			in.Description = &value
		}
	}
	if raw, ok := r.raw["amount"]; ok && !isJSONNull(raw) {
		var amount ExpenseAmount
		if err := json.Unmarshal(raw, &amount); err != nil {
			return in, fmt.Errorf("invalid amount")
		}
		in.Amount = &appdto.MoneyAmount{Amount: amount.Amount, Currency: amount.Currency}
	}
	if raw, ok := r.raw["category"]; ok && !isJSONNull(raw) {
		value, err := decodeExpenseString(raw, "category")
		if err != nil {
			return in, err
		}
		category := entity.ExpenseCategory(value)
		in.Category = &category
	}
	if raw, ok := r.raw["expenseDate"]; ok && !isJSONNull(raw) {
		value, err := decodeExpenseString(raw, "expenseDate")
		if err != nil {
			return in, err
		}
		parsed, err := parseExpenseDate(value, "expenseDate")
		if err != nil {
			return in, err
		}
		in.ExpenseDate = &parsed
	}
	if raw, ok := r.raw["paidByUserId"]; ok && !isJSONNull(raw) {
		value, err := decodeExpenseString(raw, "paidByUserId")
		if err != nil {
			return in, err
		}
		parsed, err := parseExpenseUUID(value, "paidByUserId")
		if err != nil {
			return in, err
		}
		in.PaidByUserID = &parsed
	}
	if raw, ok := r.raw["splitType"]; ok && !isJSONNull(raw) {
		value, err := decodeExpenseString(raw, "splitType")
		if err != nil {
			return in, err
		}
		splitType := entity.ExpenseSplitType(value)
		in.SplitType = &splitType
	}
	if raw, ok := r.raw["participantUserIds"]; ok {
		in.ParticipantUserIDsSet = true
		if !isJSONNull(raw) {
			var ids []string
			if err := json.Unmarshal(raw, &ids); err != nil {
				return in, fmt.Errorf("invalid participantUserIds")
			}
			parsed, err := parseExpenseUUIDs(ids, "participantUserIds")
			if err != nil {
				return in, err
			}
			in.ParticipantUserIDs = parsed
		}
	}
	if raw, ok := r.raw["customShares"]; ok {
		in.CustomSharesSet = true
		if !isJSONNull(raw) {
			var shares []ExpenseCustomAmount
			if err := json.Unmarshal(raw, &shares); err != nil {
				return in, fmt.Errorf("invalid customShares")
			}
			parsed, err := customShareInputs(shares)
			if err != nil {
				return in, err
			}
			in.CustomShares = parsed
		}
	}
	if raw, ok := r.raw["customPercentages"]; ok {
		in.CustomPercentagesSet = true
		if !isJSONNull(raw) {
			var percentages []ExpenseCustomPercentage
			if err := json.Unmarshal(raw, &percentages); err != nil {
				return in, fmt.Errorf("invalid customPercentages")
			}
			parsed, err := customPercentageInputs(percentages)
			if err != nil {
				return in, err
			}
			in.CustomPercentages = parsed
		}
	}
	if raw, ok := r.raw["linkedItinerary"]; ok {
		in.LinkedItinerarySet = true
		if !isJSONNull(raw) {
			var ref appdto.LinkedItineraryRef
			if err := json.Unmarshal(raw, &ref); err != nil {
				return in, fmt.Errorf("invalid linkedItinerary")
			}
			in.LinkedItinerary = &ref
		}
	}
	if raw, ok := r.raw["linkedRouteLegId"]; ok {
		in.LinkedRouteLegIDSet = true
		if !isJSONNull(raw) {
			value, err := decodeExpenseString(raw, "linkedRouteLegId")
			if err != nil {
				return in, err
			}
			in.LinkedRouteLegID = &value
		}
	}
	if raw, ok := r.raw["linkedAccommodation"]; ok && !isJSONNull(raw) {
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return in, fmt.Errorf("invalid linkedAccommodation")
		}
		in.LinkedAccommodation = &value
	}
	if raw, ok := r.raw["notes"]; ok {
		if isJSONNull(raw) {
			in.ClearNotes = true
		} else {
			value, err := decodeExpenseString(raw, "notes")
			if err != nil {
				return in, err
			}
			in.Notes = &value
		}
	}
	if raw, ok := r.raw["metadata"]; ok && !isJSONNull(raw) {
		var metadata map[string]any
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return in, fmt.Errorf("invalid metadata")
		}
		in.Metadata = metadata
	}
	return in, nil
}

type MarkSettlementPaid struct {
	Notes *string `json:"notes"`
}

func (r MarkSettlementPaid) ToInput() appdto.MarkSettlementPaidInput {
	return appdto.MarkSettlementPaidInput{Notes: r.Notes}
}

func customShareInputs(shares []ExpenseCustomAmount) ([]appdto.ExpenseCustomAmount, error) {
	out := make([]appdto.ExpenseCustomAmount, 0, len(shares))
	for _, share := range shares {
		userID, err := parseExpenseUUID(share.UserID, "customShares.userId")
		if err != nil {
			return nil, err
		}
		out = append(out, appdto.ExpenseCustomAmount{
			UserID:   userID,
			Amount:   share.Amount,
			Currency: share.Currency,
		})
	}
	return out, nil
}

func customPercentageInputs(items []ExpenseCustomPercentage) ([]appdto.ExpenseCustomPercentage, error) {
	out := make([]appdto.ExpenseCustomPercentage, 0, len(items))
	for _, item := range items {
		userID, err := parseExpenseUUID(item.UserID, "customPercentages.userId")
		if err != nil {
			return nil, err
		}
		out = append(out, appdto.ExpenseCustomPercentage{
			UserID:     userID,
			Percentage: item.Percentage,
		})
	}
	return out, nil
}

func parseExpenseUUIDs(values []string, field string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		parsed, err := parseExpenseUUID(value, field)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parseExpenseUUID(value string, field string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", field)
	}
	return parsed, nil
}

func parseExpenseDate(value string, field string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s", field)
	}
	return parsed, nil
}

func decodeExpenseString(raw json.RawMessage, field string) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("invalid %s", field)
	}
	return value, nil
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
