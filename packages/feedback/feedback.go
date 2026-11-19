package feedback

// Feedback is the container binding used by facades.Feedback().
// Prefer NewService() / contracts.Service in application code.
type Feedback struct {
	*Service
}

func NewFeedback() *Feedback {
	return &Feedback{Service: NewService()}
}
