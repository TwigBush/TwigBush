package policy

type Decision string

const (
	Allow  Decision = "allow"
	StepUp Decision = "step_up"
	Deny   Decision = "deny"
)

type Checker interface {
	Evaluate(human string, agent string, action string, object string) (Decision, error)
}
