package dto

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/safeconv"
)

func int4PtrArg(value *int, name string) (pgtype.Int4, error) {
	if value == nil {
		return pgtype.Int4{Valid: false}, nil
	}
	converted, err := safeconv.IntToInt32(*value, name)
	if err != nil {
		return pgtype.Int4{}, err
	}
	return pgtype.Int4{Int32: converted, Valid: true}, nil
}
