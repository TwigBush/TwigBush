package authz

import "context"

type Decision struct {
	Allowed bool
	Reason  string
}

type Request struct {
	Subject  string         // e.g. end-user sub or client key id
	Relation string         // e.g. "checkout", "pay", "quote"
	Object   string         // e.g. "cart:1234" or "merchant:schnucks"
	Context  map[string]any // optional: constraints passed to conditions
}

type Authorizer interface {
	Check(ctx context.Context, req Request) (Decision, error)
}
