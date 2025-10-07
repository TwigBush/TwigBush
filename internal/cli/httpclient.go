package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func httpDoJSON(method, url string, body []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if showCurl {
		fmt.Println(curlFor(method, url, body, headers))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

func curlFor(method, url string, body []byte, headers map[string]string) string {
	sb := &bytes.Buffer{}
	fmt.Fprintf(sb, "curl -i -X %s '%s'", method, url)
	for k, v := range headers {
		fmt.Fprintf(sb, " -H %q", fmt.Sprintf("%s: %s", k, v))
	}
	if len(body) > 0 {
		tmp := ".curl-body.json"
		_ = os.WriteFile(tmp, body, 0o600)
		fmt.Fprintf(sb, " --data-binary @%s", tmp)
	}
	return sb.String()
}

func printJSON(b []byte) error {
	var any interface{}
	if err := json.Unmarshal(b, &any); err != nil {
		// not JSON, print raw
		fmt.Println(string(b))
		return nil
	}
	enc, _ := json.MarshalIndent(any, "", "  ")
	fmt.Println(string(enc))
	return nil
}
