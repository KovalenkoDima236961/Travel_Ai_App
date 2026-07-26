package postgres

import "github.com/KovalenkoDima236961/Travel_Ai_App/internal/safeconv"

func limitArg(value int) (uint64, error) {
	return safeconv.NonNegativeIntToUint64(value, "limit")
}

func offsetArg(value int) (uint64, error) {
	return safeconv.NonNegativeIntToUint64(value, "offset")
}
