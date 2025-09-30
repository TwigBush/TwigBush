package authz

import "context"

type Mock struct {
	AlwaysAllow bool
}

func (m *Mock) Check(ctx context.Context, req Request) (Decision, error) {
	if m.AlwaysAllow {
		return Decision{Allowed: true}, nil
	}
	return Decision{Allowed: false, Reason: "mock_deny"}, nil
}
