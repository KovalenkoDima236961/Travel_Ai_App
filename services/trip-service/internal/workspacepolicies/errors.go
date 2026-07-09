package workspacepolicies

type BlockingViolationError struct {
	Evaluation Evaluation
}

func (e *BlockingViolationError) Error() string {
	return "this trip has blocking workspace policy violations"
}
