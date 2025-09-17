package jwks

import (
	"encoding/json"
	"net/http"
)

func Serve(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"keys": []any{},
	})
}
