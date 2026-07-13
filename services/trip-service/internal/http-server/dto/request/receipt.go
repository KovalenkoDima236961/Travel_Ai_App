package request

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type ExtractReceipt struct {
	Provider string `json:"provider"`
}

func (r ExtractReceipt) ToInput() (appdto.ExtractReceiptInput, error) {
	provider := strings.TrimSpace(r.Provider)
	if provider == "" {
		return appdto.ExtractReceiptInput{}, nil
	}
	value := entity.ReceiptOCRProvider(provider)
	switch value {
	case entity.ReceiptOCRProviderMock, entity.ReceiptOCRProviderLocal, entity.ReceiptOCRProviderManual:
		return appdto.ExtractReceiptInput{Provider: &value}, nil
	default:
		return appdto.ExtractReceiptInput{}, apperrs.NewInvalidInput("unsupported OCR provider")
	}
}

type AttachReceipt struct {
	ReceiptID string `json:"receiptId"`
}

func (r AttachReceipt) ToInput() (appdto.AttachReceiptInput, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(r.ReceiptID))
	if err != nil {
		return appdto.AttachReceiptInput{}, apperrs.NewInvalidInput("invalid receiptId")
	}
	return appdto.AttachReceiptInput{ReceiptID: parsed}, nil
}

func DecodeExtractReceipt(raw []byte) (appdto.ExtractReceiptInput, error) {
	if len(raw) == 0 {
		return appdto.ExtractReceiptInput{}, nil
	}
	var req ExtractReceipt
	if err := json.Unmarshal(raw, &req); err != nil {
		return appdto.ExtractReceiptInput{}, err
	}
	return req.ToInput()
}
