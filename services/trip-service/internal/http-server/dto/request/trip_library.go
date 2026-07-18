package request

import appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"

// ArchiveTrip deliberately accepts one optional short reason. Free text is
// stored with the trip but is never copied to activity metadata or logs.
type ArchiveTrip struct {
	Reason string `json:"reason" validate:"omitempty,max=500"`
}

func (r ArchiveTrip) ToInput() appdto.ArchiveTripInput {
	return appdto.ArchiveTripInput{Reason: r.Reason}
}
