package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type CreateTripTravelerInput struct {
	Name         string
	Email        *string
	LinkedUserID *uuid.UUID
	Role         entity.TripTravelerRole
}

type UpdateTripTravelerInput struct {
	Name  *string
	Email *string
	Role  *entity.TripTravelerRole
}

type UpdateItemCostSplitInput struct {
	ExpectedItineraryRevision *int
	Split                     *aggregate.CostSplitRule
}

type UpdateAccommodationCostSplitInput struct {
	Split *aggregate.CostSplitRule
}

type CostSplittingSummary struct {
	TripID           uuid.UUID                `json:"tripId"`
	Currency         string                   `json:"currency"`
	GeneratedAt      time.Time                `json:"generatedAt"`
	Summary          CostSplittingTotals      `json:"summary"`
	Travelers        []TravelerCostAllocation `json:"travelers"`
	UnassignedCosts  []UnassignedCost         `json:"unassignedCosts"`
	ByCategory       []CostSplitCategoryTotal `json:"byCategory"`
	ByDay            []CostSplitDayTotal      `json:"byDay"`
	Warnings         []string                 `json:"warnings"`
	ExchangeRateInfo *budget.ExchangeRateInfo `json:"exchangeRateInfo,omitempty"`
}

type CostSplittingTotals struct {
	TravelerCount        int     `json:"travelerCount"`
	EstimatedTotal       float64 `json:"estimatedTotal"`
	AllocatedTotal       float64 `json:"allocatedTotal"`
	UnassignedTotal      float64 `json:"unassignedTotal"`
	MissingEstimateCount int     `json:"missingEstimateCount"`
	DefaultSplitCount    int     `json:"defaultSplitCount"`
	InvalidSplitCount    int     `json:"invalidSplitCount"`
	ConvertedItemCount   int     `json:"convertedItemCount"`
	UnconvertedItemCount int     `json:"unconvertedItemCount"`
}

type TravelerCostAllocation struct {
	TravelerID        uuid.UUID                `json:"travelerId"`
	Name              string                   `json:"name"`
	Email             *string                  `json:"email,omitempty"`
	LinkedUserID      *uuid.UUID               `json:"linkedUserId,omitempty"`
	Role              entity.TripTravelerRole  `json:"role"`
	AllocatedTotal    float64                  `json:"allocatedTotal"`
	PercentageOfTotal float64                  `json:"percentageOfTotal"`
	ByCategory        []CostSplitCategoryTotal `json:"byCategory"`
	ByDay             []CostSplitDayTotal      `json:"byDay"`
	Items             []TravelerAllocatedItem  `json:"items"`
}

type TravelerAllocatedItem struct {
	Type                 string  `json:"type"`
	DayNumber            *int    `json:"dayNumber,omitempty"`
	ItemIndex            *int    `json:"itemIndex,omitempty"`
	Name                 string  `json:"name"`
	Category             string  `json:"category"`
	AllocatedAmount      float64 `json:"allocatedAmount"`
	OriginalCostAmount   float64 `json:"originalCostAmount"`
	OriginalCostCurrency string  `json:"originalCostCurrency"`
	SplitType            string  `json:"splitType"`
	RuleSource           string  `json:"ruleSource"`
}

type UnassignedCost struct {
	Type      string  `json:"type"`
	DayNumber *int    `json:"dayNumber,omitempty"`
	ItemIndex *int    `json:"itemIndex,omitempty"`
	Name      string  `json:"name"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Reason    string  `json:"reason"`
}

type CostSplitCategoryTotal struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

type CostSplitDayTotal struct {
	DayNumber int     `json:"dayNumber"`
	Amount    float64 `json:"amount"`
}
