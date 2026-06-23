package dto

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// UUIDArg encodes a UUID for use in query arguments.
func UUIDArg(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromPgUUID(p pgtype.UUID) uuid.UUID {
	return uuid.UUID(p.Bytes)
}

func toPgTextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromPgTextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	value := t.String
	return &value
}

func toPgNumericPtr(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	hundredths := big.NewInt(int64(math.Round(*f * 100)))
	return pgtype.Numeric{Int: hundredths, Exp: -2, Valid: true}
}

func fromPgNumericPtr(n pgtype.Numeric) *float64 {
	if !n.Valid || n.Int == nil {
		return nil
	}
	rat := new(big.Rat).SetInt(n.Int)
	if n.Exp != 0 {
		scale := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs32(n.Exp))), nil))
		if n.Exp > 0 {
			rat.Mul(rat, scale)
		} else {
			rat.Quo(rat, scale)
		}
	}
	v, _ := rat.Float64()
	return &v
}

func marshalStringArray(values []string) ([]byte, error) {
	if values == nil {
		values = []string{}
	}
	b, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshal string array: %w", err)
	}
	return b, nil
}

func unmarshalStringArray(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("unmarshal string array: %w", err)
	}
	if values == nil {
		return []string{}, nil
	}
	return values, nil
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
