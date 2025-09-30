package authz

import (
	"context"
	"fmt"

	fga "github.com/openfga/go-sdk/client"
)

type OpenFGA struct {
	c       *fga.OpenFgaClient
	modelID string
}

type OpenFGAConfig struct {
	APIURL   string
	StoreID  string
	APIToken string // optional
	ModelID  string // optional but recommended in prod
}

func NewOpenFGA(cfg OpenFGAConfig) (*OpenFGA, error) {
	conf := &fga.ClientConfiguration{
		ApiUrl:  cfg.APIURL,
		StoreId: cfg.StoreID, // omit when creating/listing stores
	}

	// Pin a specific auth model if provided
	if cfg.ModelID != "" {
		conf.AuthorizationModelId = cfg.ModelID
	}

	client, err := fga.NewSdkClient(conf)
	if err != nil {
		return nil, fmt.Errorf("openfga_client_init: %w", err)
	}

	var modelPtr string
	if cfg.ModelID != "" {
		modelPtr = cfg.ModelID
	}

	return &OpenFGA{
		c:       client,
		modelID: modelPtr,
	}, nil
}

func (o *OpenFGA) Check(ctx context.Context, req Request) (Decision, error) {
	checkReq := fga.ClientCheckRequest{
		User:     req.Subject,  // e.g. "user:alice" or "client:bot-42"
		Relation: req.Relation, // e.g. "checkout"
		Object:   req.Object,   // e.g. "merchant:schnucks" or "basket:abc"
	}

	// Basic check
	resp, err := o.c.Check(ctx).Body(checkReq).Execute()
	if err != nil {
		return Decision{}, fmt.Errorf("fga_check_error: %w", err)
	}

	if resp.Allowed != nil && *resp.Allowed {
		return Decision{Allowed: true}, nil
	}
	return Decision{Allowed: false, Reason: "policy_denied"}, nil
}
