package gnap

import "time"

type Continue struct {
	AccessToken string `json:"access_token"`
	URI         string `json:"uri"`
	Wait        int    `json:"wait"` // seconds to poll before calling /continue
}

type UserCode struct {
	Code string `json:"code"`
	URI  string `json:"uri"`
}

type InteractOut struct {
	Expires  time.Time `json:"expires"`
	UserCode UserCode  `json:"user_code"`
}

type GrantResponse struct {
	Continue Continue    `json:"continue"`
	Interact InteractOut `json:"interact"`
}
