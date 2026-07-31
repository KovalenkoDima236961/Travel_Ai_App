package search

func resultTypeFilters(types []ResultType) (map[ResultType]struct{}, error) {
	if len(types) == 0 {
		return nil, nil
	}
	filters := make(map[ResultType]struct{}, len(types))
	for _, resultType := range types {
		if !knownResultType(resultType) {
			return nil, ErrInvalidFilter
		}
		filters[resultType] = struct{}{}
	}
	return filters, nil
}

func typeAllowed(filters map[ResultType]struct{}, resultType ResultType) bool {
	if len(filters) == 0 {
		return true
	}
	_, ok := filters[resultType]
	return ok
}

func anyTypeAllowed(filters map[ResultType]struct{}, resultTypes ...ResultType) bool {
	if len(filters) == 0 {
		return true
	}
	for _, resultType := range resultTypes {
		if _, ok := filters[resultType]; ok {
			return true
		}
	}
	return false
}

func knownResultType(resultType ResultType) bool {
	switch resultType {
	case ResultTypeTrip,
		ResultTypeWorkspace,
		ResultTypeTemplate,
		ResultTypeItineraryItem,
		ResultTypeRouteStop,
		ResultTypeRouteLeg,
		ResultTypeTransportOption,
		ResultTypeExpense,
		ResultTypeReceipt,
		ResultTypeChecklistItem,
		ResultTypeReminder,
		ResultTypePoll,
		ResultTypeCollaborator,
		ResultTypeNotification,
		ResultTypeSetting,
		ResultTypeCommand,
		ResultTypeOpsPage:
		return true
	default:
		return false
	}
}
