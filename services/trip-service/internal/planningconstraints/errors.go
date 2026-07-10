package planningconstraints

type BlockingError struct {
	Constraints PlanningConstraints
}

func (e *BlockingError) Error() string {
	return "planning constraints contain blocking issues"
}

func NewBlockingError(constraints PlanningConstraints) *BlockingError {
	return &BlockingError{Constraints: constraints}
}
