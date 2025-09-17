package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"regexp"
	"time"

	"github.com/TwigBush/gnap-go/internal/gnap"
	"github.com/TwigBush/gnap-go/internal/httpx"
)

var userCodeRe = regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{4}$`)

type DeviceHandler struct {
	Store gnap.Store
}

func NewDeviceHandler(store gnap.Store) *DeviceHandler {
	return &DeviceHandler{Store: store}
}

type verifyRequest struct {
	UserCode string `json:"user_code"`
}

// GrantStateJSON mirrors your Java DTO exactly.
type GrantStateJSON struct {
	ID                string            `json:"id"`
	Client            gnap.Client       `json:"client"`
	RequestedAccess   []gnap.AccessItem `json:"requested_access"`
	Status            gnap.GrantStatus  `json:"status"`
	ContinuationToken string            `json:"continuation_token"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
	ExpiresAt         string            `json:"expires_at"`
	InteractionNonce  string            `json:"interaction_nonce"`
	UserCode          string            `json:"user_code"`
	Subject           string            `json:"subject"`
	ApprovedAccess    []gnap.AccessItem `json:"approved_access"`
	Locations         []string          `json:"locations"`
}

// POST /device/verify (JSON) -> JSON GrantState (Java shape)
func (h *DeviceHandler) VerifyJSON(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if !userCodeRe.MatchString(req.UserCode) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid user_code format")
		return
	}

	grant, ok := h.Store.FindGrantByUserCodePending(r.Context(), req.UserCode)
	if !ok || grant == nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid or expired code. Please try again.")
		return
	}

	out := mapGrantStateToJavaJSON(grant)
	httpx.WriteJSON(w, http.StatusOK, out)
}

func mapGrantStateToJavaJSON(g *gnap.GrantState) GrantStateJSON {
	return GrantStateJSON{
		ID:                g.ID,
		Client:            g.Client,
		RequestedAccess:   g.RequestedAccess,
		Status:            g.Status,
		ContinuationToken: g.ContinuationToken,
		CreatedAt:         formatRFC3339(g.CreatedAt),
		UpdatedAt:         formatRFC3339(g.UpdatedAt),
		ExpiresAt:         formatRFC3339(g.ExpiresAt),
		//InteractionNonce:  deref(g.InteractionNonce),
		UserCode:       deref(g.UserCode),
		Subject:        deref(g.Subject),
		ApprovedAccess: g.ApprovedAccess,
		Locations:      unmarshalLocations(g.Locations),
	}
}

func formatRFC3339(t time.Time) string {
	// Match Java string timestamps; RFC3339 is a sane default.
	return t.UTC().Format(time.RFC3339Nano)
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func unmarshalLocations(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var xs []string
	if err := json.Unmarshal(raw, &xs); err != nil {
		return nil
	}
	return xs
}

func (h *DeviceHandler) ConsentForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		deviceError(w, "Invalid form submission.")
		return
	}
	grantID := r.Form.Get("grant_id")
	decision := r.Form.Get("decision")

	// lookup grant (expire-on-read handled in store.GetGrant)
	g, ok := h.Store.GetGrant(r.Context(), grantID)
	if !ok || g == nil {
		deviceError(w, "Grant not found")
		return
	}

	switch decision {
	case "approve":
		// Approve with requested_access and a subject "user:device"
		_, err := h.Store.ApproveGrant(r.Context(), grantID, g.RequestedAccess, "user:device")
		if err != nil {
			deviceError(w, httpx.SafeErrMsg(err)) // simple stringify
			return
		}
		deviceSuccess(w)
	default:
		_, err := h.Store.DenyGrant(r.Context(), grantID)
		if err != nil {
			deviceError(w, httpx.SafeErrMsg(err))
			return
		}
		deviceDenied(w)
	}
}

// POST /device/verify (form-urlencoded) -> HTML consent screen
func (h *DeviceHandler) VerifyForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		deviceError(w, "Invalid form submission. Please try again.")
		return
	}
	userCode := r.Form.Get("user_code")
	if !userCodeRe.MatchString(userCode) {
		deviceError(w, "Invalid or expired code. Please try again.")
		return
	}

	grant, ok := h.Store.FindGrantByUserCodePending(r.Context(), userCode)
	if !ok || grant == nil {
		deviceError(w, "Invalid or expired code. Please try again.")
		return
	}

	// Render the consent screen for this grant
	consentScreen(w, grant)
}

// GET /device – renders a pretty page with inline validation
func (h *DeviceHandler) Page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := devicePageTmpl.Execute(w, nil); err != nil {
		http.Error(w, "template render error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

var devicePageTmpl = template.Must(template.New("devicePage").Parse(`
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Device Verification</title>
<style>
  :root{
    --bg:#0b0f14; --card:#111827; --muted:#9ca3af; --accent:#2563eb; --accent-2:#60a5fa;
    --error:#ef4444; --ok:#10b981; --text:#e5e7eb
  }
  *{box-sizing:border-box}
  html, body { height: 100%; }
  body{
    margin:0; color:var(--text);
    font-family: ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;
    position: relative; min-height: 100vh;
  }
  /* Fixed full-viewport gradient background (prevents cut-off and plays nice with blur) */
  body::before{
    content:""; position:fixed; inset:0; z-index:-1;
    background: linear-gradient(180deg, #0b0f14 0%, #0f172a 100%);
    background-attachment: fixed;
  }

  .wrap{max-width:520px;margin:6vh auto;padding:24px}
  .card{
    background:rgba(17,24,39,.75); backdrop-filter:blur(8px);
    border:1px solid #1f2937; border-radius:16px; padding:28px;
    box-shadow:0 10px 30px rgba(0,0,0,.35)
  }
  h1{margin:0 0 6px;font-size:22px;letter-spacing:.2px}
  p{margin:0 0 20px;color:var(--muted);line-height:1.5}
  form{display:grid;gap:14px}
  label{font-size:13px;color:#cbd5e1}
  .row{display:flex;gap:10px;align-items:center}

  input[type="text"]{
    width:100%;
    font:600 18px/1.2 ui-monospace,SFMono-Regular,Menlo,monospace;
    color:#e2e8f0; background:#0b1220; border:1px solid #1f2937; border-radius:12px;
    padding:14px 14px; letter-spacing:1px; outline:none;
    transition:border .15s, box-shadow .15s, background .15s, transform .08s;
    text-transform:uppercase;
  }
  input[type="text"]:focus{border-color:#334155;box-shadow:0 0 0 3px rgba(37,99,235,.25)}
  .hint{font-size:12px;color:var(--muted)}
  .hint.ok{color:var(--ok)}
  .hint.err{color:var(--error)}

  button{
    border:0;border-radius:12px;padding:12px 16px;font-weight:700;cursor:pointer;
    background:linear-gradient(135deg,var(--accent),var(--accent-2));color:white;
    box-shadow:0 6px 16px rgba(37,99,235,.35);
    transition:transform .06s ease, filter .15s ease, box-shadow .15s ease;
  }
  button:disabled{filter:grayscale(.6) brightness(.8);cursor:not-allowed;box-shadow:none}
  button:active{transform:translateY(1px)}
  .footer{margin-top:10px;font-size:12px;color:#94a3b8}
  .errbox{display:none;margin:-6px 0 4px;font-size:12px;color:var(--error)}
</style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <h1>Enter your user code</h1>
      <p>Type the 8-character code shown on your device. It looks like <code>ABCD-1234</code>.</p>

      <form id="verifyForm" method="post" action="/device/verify" novalidate>
        <label for="user_code">User code</label>
        <div class="row">
          <input id="user_code" name="user_code" type="text" inputmode="latin"
                 autocomplete="one-time-code" placeholder="ABCD-1234"
                 maxlength="9" pattern="^[A-Z0-9]{4}-[A-Z0-9]{4}$" required>
          <button id="submitBtn" type="submit" disabled>Verify</button>
        </div>
        <div id="msg" class="hint">Format: <b>ABCD-1234</b> (A–Z, 0–9)</div>
        <div id="err" class="errbox">Invalid code. Please use the format ABCD-1234.</div>
      </form>

      <div class="footer">Powered by TwigBush GNAP</div>
    </div>
  </div>

<script>
(function() {
  const input = document.getElementById('user_code');
  const btn   = document.getElementById('submitBtn');
  const msg   = document.getElementById('msg');
  const err   = document.getElementById('err');
  const re    = /^[A-Z0-9]{4}-[A-Z0-9]{4}$/;

  function normalize(v) {
    v = (v || '').toUpperCase().replace(/[^A-Z0-9]/g,'');
    if (v.length > 4) v = v.slice(0,4) + '-' + v.slice(4,8);
    return v.slice(0,9);
  }

  function validate(v) {
    const ok = re.test(v);
    btn.disabled = !ok;
    msg.className = 'hint ' + (ok ? 'ok' : '');
    msg.textContent = ok ? 'Looks good. You can verify now.' : 'Format: ABCD-1234 (A–Z, 0–9)';
    err.style.display = 'none';
    return ok;
  }

  input.addEventListener('input', () => {
    const cur = input.value;
    const norm = normalize(cur);
    if (cur !== norm) {
      input.value = norm;
      input.setSelectionRange(norm.length, norm.length);
    }
    validate(input.value);
  });

  input.addEventListener('blur', () => validate(input.value));

  input.addEventListener('paste', (e) => {
    e.preventDefault();
    const text = (e.clipboardData || window.clipboardData).getData('text');
    input.value = normalize(text);
    validate(input.value);
  });

  document.getElementById('verifyForm').addEventListener('submit', (e) => {
    const ok = validate(input.value);
    if (!ok) {
      e.preventDefault();
      err.style.display = 'block';
      return;
    }
    btn.disabled = true;
    btn.textContent = 'Verifying…';
  });

  input.focus();
})();
</script>
</body>
</html>
`))

// Reuse existing deviceErrorTmpl if you already have it.
// Otherwise:
var deviceSuccessTmpl = template.Must(template.New("success").Parse(`
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Consent Success</title>
<style>
  :root{
    --bg:#0b0f14; --card:#111827; --muted:#9ca3af; --accent:#2563eb; --accent-2:#60a5fa;
    --ok:#10b981; --text:#e5e7eb
  }
  *{box-sizing:border-box}
  html, body { height: 100%; }
  body{
    margin:0; color:var(--text);
    font-family: ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;
    position: relative; min-height: 100vh;
  }
  body::before{
    content:""; position:fixed; inset:0; z-index:-1;
    background: linear-gradient(180deg, #0b0f14 0%, #0f172a 100%);
    background-attachment: fixed;
  }
  .wrap{max-width:520px;margin:10vh auto;padding:24px}
  .card{
    background:rgba(17,24,39,.75); backdrop-filter:blur(8px);
    border:1px solid #1f2937; border-radius:16px; padding:32px;
    box-shadow:0 10px 30px rgba(0,0,0,.35); text-align:center
  }
  h1{margin:0 0 10px;font-size:24px;letter-spacing:.2px}
  p{margin:0 0 18px;color:var(--muted);line-height:1.6}
  .icon{
    width:64px;height:64px;margin:0 auto 14px; border-radius:50%;
    background:rgba(16,185,129,.12); display:grid; place-items:center;
    border:1px solid rgba(16,185,129,.25)
  }
  .check{
    width:30px;height:30px; display:block; position:relative;
  }
  .check::before,.check::after{
    content:""; position:absolute; background:var(--ok); border-radius:2px;
  }
  .check::before{
    width:6px;height:16px; left:10px; top:7px; transform:rotate(45deg);
  }
  .check::after{
    width:6px;height:28px; left:18px; top:-1px; transform:rotate(-45deg);
  }
  .actions{margin-top:8px}
  .btn{
    display:inline-block; border:0; border-radius:12px; padding:12px 16px;
    font-weight:700; cursor:pointer; color:white;
    background:linear-gradient(135deg,var(--accent),var(--accent-2));
    box-shadow:0 6px 16px rgba(37,99,235,.35);
    transition:transform .06s ease, filter .15s ease, box-shadow .15s ease;
    text-decoration:none;
  }
  .btn:active{transform:translateY(1px)}
  .meta{margin-top:12px; font-size:12px; color:#94a3b8}
</style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <div class="icon"><span class="check" aria-hidden="true"></span></div>
      <h1>Consent recorded</h1>
      <p>You may return to your device to continue.</p>
      <div class="actions">
        <a class="btn" href="/device">Back to code entry</a>
      </div>
      <div class="meta">TwigBush GNAP</div>
    </div>
  </div>
</body>
</html>
`))

var deviceDeniedTmpl = template.Must(template.New("denied").Parse(`
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Consent Denied</title>
<style>
  :root{
    --bg:#0b0f14; --card:#111827; --muted:#9ca3af; --accent:#2563eb; --accent-2:#60a5fa;
    --warn:#f59e0b; --text:#e5e7eb
  }
  *{box-sizing:border-box}
  html, body { height: 100%; }
  body{
    margin:0; color:var(--text);
    font-family: ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;
    position: relative; min-height: 100vh;
  }
  body::before{
    content:""; position:fixed; inset:0; z-index:-1;
    background: linear-gradient(180deg, #0b0f14 0%, #0f172a 100%);
    background-attachment: fixed;
  }
  .wrap{max-width:520px;margin:10vh auto;padding:24px}
  .card{
    background:rgba(17,24,39,.75); backdrop-filter:blur(8px);
    border:1px solid #1f2937; border-radius:16px; padding:32px;
    box-shadow:0 10px 30px rgba(0,0,0,.35); text-align:center
  }
  h1{margin:0 0 10px;font-size:24px;letter-spacing:.2px}
  p{margin:0 0 18px;color:var(--muted);line-height:1.6}
  .icon{
    width:64px;height:64px;margin:0 auto 14px; border-radius:50%;
    background:rgba(245,158,11,.12); display:grid; place-items:center;
    border:1px solid rgba(245,158,11,.25)
  }
  .minus{
    position:relative;width:30px;height:30px;display:block
  }
  .minus::before{
    content:""; position:absolute; left:3px; right:3px; top:12px; height:6px;
    background:var(--warn); border-radius:3px;
  }
  .actions{margin-top:8px}
  .btn{
    display:inline-block; border:0; border-radius:12px; padding:12px 16px;
    font-weight:700; cursor:pointer; color:white; text-decoration:none;
    background:linear-gradient(135deg,var(--accent),var(--accent-2));
    box-shadow:0 6px 16px rgba(37,99,235,.35);
    transition:transform .06s ease, filter .15s ease, box-shadow .15s ease;
  }
  .btn:active{transform:translateY(1px)}
  .meta{margin-top:12px; font-size:12px; color:#94a3b8}
</style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <div class="icon"><span class="minus" aria-hidden="true"></span></div>
      <h1>Consent denied</h1>
      <p>You chose to deny access. You can return to your device or enter a different code.</p>
      <div class="actions">
        <a class="btn" href="/device">Back to code entry</a>
      </div>
      <div class="meta">TwigBush GNAP</div>
    </div>
  </div>
</body>
</html>
`))

var deviceErrorTmpl = template.Must(template.New("deviceErr").Parse(`
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Device Verification</title>
<style>
  :root{
    --bg:#0b0f14; --card:#111827; --muted:#9ca3af; --accent:#2563eb; --accent-2:#60a5fa;
    --error:#ef4444; --text:#e5e7eb
  }
  *{box-sizing:border-box}
  html, body { height: 100%; }
  body{
    margin:0; color:var(--text);
    font-family: ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;
    position: relative; min-height: 100vh;
  }
  body::before{
    content:""; position:fixed; inset:0; z-index:-1;
    background: linear-gradient(180deg, #0b0f14 0%, #0f172a 100%);
    background-attachment: fixed;
  }
  .wrap{max-width:520px;margin:10vh auto;padding:24px}
  .card{
    background:rgba(17,24,39,.75); backdrop-filter:blur(8px);
    border:1px solid #1f2937; border-radius:16px; padding:32px;
    box-shadow:0 10px 30px rgba(0,0,0,.35); text-align:center
  }
  h1{margin:0 0 10px;font-size:24px;letter-spacing:.2px}
  p{margin:0 0 18px;color:var(--muted);line-height:1.6}
  .icon{
    width:64px;height:64px;margin:0 auto 14px; border-radius:50%;
    background:rgba(239,68,68,.12); display:grid; place-items:center;
    border:1px solid rgba(239,68,68,.25)
  }
  .xmark{position:relative;width:30px;height:30px;display:block}
  .xmark::before,.xmark::after{
    content:""; position:absolute; left:12px; top:0; width:6px; height:30px;
    background:var(--error); border-radius:2px;
  }
  .xmark::before{ transform:rotate(45deg); }
  .xmark::after{ transform:rotate(-45deg); }

  .err{color:#fecaca; background:rgba(239,68,68,.12); border:1px solid rgba(239,68,68,.25);
       padding:10px 12px; border-radius:10px; margin:0 auto 12px; display:inline-block}

  .actions{margin-top:8px}
  .btn{
    display:inline-block; border:0; border-radius:12px; padding:12px 16px;
    font-weight:700; cursor:pointer; color:white; text-decoration:none;
    background:linear-gradient(135deg,var(--accent),var(--accent-2));
    box-shadow:0 6px 16px rgba(37,99,235,.35);
    transition:transform .06s ease, filter .15s ease, box-shadow .15s ease;
  }
  .btn:active{transform:translateY(1px)}
  .meta{margin-top:12px; font-size:12px; color:#94a3b8}
</style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <div class="icon"><span class="xmark" aria-hidden="true"></span></div>
      <h1>Device Verification</h1>
      <p class="err">{{ .Message }}</p>
      <div class="actions">
        <a class="btn" href="/device">Back to code entry</a>
      </div>
      <div class="meta">TwigBush GNAP</div>
    </div>
  </div>
</body>
</html>
`))

func deviceError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK) // many device flows still return 200 with an error message
	_ = deviceErrorTmpl.Execute(w, struct{ Message string }{Message: msg})
}

func deviceSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = deviceSuccessTmpl.Execute(w, nil)
}
func deviceDenied(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = deviceDeniedTmpl.Execute(w, nil)
}

var consentScreenTmpl = template.Must(template.New("consent").Parse(`
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Consent Required</title>
<style>
  :root{
    --bg:#0b0f14; --card:#111827; --muted:#9ca3af; --accent:#2563eb; --accent-2:#60a5fa;
    --error:#ef4444; --ok:#10b981; --text:#e5e7eb
  }
  *{box-sizing:border-box}
  html, body { height: 100%; }
  body{
    margin:0; color:var(--text);
    font-family: ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;
    position: relative; min-height: 100vh;
  }
  body::before{
    content:""; position:fixed; inset:0; z-index:-1;
    background: linear-gradient(180deg, #0b0f14 0%, #0f172a 100%);
    background-attachment: fixed;
  }
  .wrap{max-width:720px;margin:6vh auto;padding:24px}
  .card{
    background:rgba(17,24,39,.75); backdrop-filter:blur(8px);
    border:1px solid #1f2937; border-radius:16px; padding:28px;
    box-shadow:0 10px 30px rgba(0,0,0,.35)
  }
  h1{margin:0 0 10px;font-size:22px;letter-spacing:.2px}
  p{margin:0 0 18px;color:var(--muted);line-height:1.6}
  .list{margin:12px 0 18px; padding:0; list-style:none; display:grid; gap:12px}
  .item{
    border:1px solid #273244; border-radius:12px; padding:14px 16px;
    background:#0b1220;
  }
  .kv{display:flex; gap:8px; flex-wrap:wrap; font-size:14px}
  .kv b{color:#cbd5e1; min-width:110px}
  .chips{display:flex; flex-wrap:wrap; gap:8px; margin-top:6px}
  .chip{
    font:600 12px/1 ui-monospace,SFMono-Regular,Menlo,monospace;
    padding:6px 8px; border-radius:10px; background:#0c1627; border:1px solid #1f2a3d; color:#e2e8f0
  }
  .actions{display:flex; gap:10px; margin-top:10px}
  button{
    border:0;border-radius:12px;padding:12px 16px;font-weight:700;cursor:pointer;
    background:linear-gradient(135deg,var(--accent),var(--accent-2));color:white;
    box-shadow:0 6px 16px rgba(37,99,235,.35);
    transition:transform .06s ease, filter .15s ease, box-shadow .15s ease;
  }
  button:active{transform:translateY(1px)}
  .deny{
    background:linear-gradient(135deg,#374151,#111827); color:#e5e7eb;
    border:1px solid #374151; box-shadow:none;
  }
  .meta{font-size:12px;color:#94a3b8;margin-top:6px}
</style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <h1>Grant Consent</h1>
      <p>Device <strong>{{ .UserCode }}</strong> is requesting access. Review and approve or deny.</p>

      {{ if .Requested }}
      <ul class="list">
        {{ range .Requested }}
        <li class="item">
          <div class="kv"><b>Type</b><span>{{ .Type }}</span></div>
          {{ if .ResourceID }}<div class="kv"><b>Resource</b><span>{{ .ResourceID }}</span></div>{{ end }}
          {{ if .Locations }}
            <div class="kv"><b>Locations</b>
              <span>
                {{ range $i, $loc := .Locations }}{{ if $i }}, {{ end }}{{ $loc }}{{ end }}
              </span>
            </div>
          {{ end }}
          {{ if .Actions }}
            <div class="kv"><b>Actions</b></div>
            <div class="chips">
              {{ range .Actions }}<span class="chip">{{ . }}</span>{{ end }}
            </div>
          {{ end }}
          {{ if .Constraints }}
            <div class="kv"><b>Constraints</b></div>
            <div class="chips">
              {{ range $k, $v := .Constraints }}<span class="chip">{{ $k }}={{ $v }}</span>{{ end }}
            </div>
          {{ end }}
        </li>
        {{ end }}
      </ul>
      {{ end }}

      <form method="post" action="/device/consent" class="actions">
        <input type="hidden" name="grant_id" value="{{ .GrantID }}">
        <button type="submit" name="decision" value="approve">Approve</button>
        <button class="deny" type="submit" name="decision" value="deny">Deny</button>
      </form>

      <div class="meta">Instance: {{ .GrantID }}</div>
    </div>
  </div>
</body>
</html>
`))

func consentScreen(w http.ResponseWriter, g *gnap.GrantState) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = consentScreenTmpl.Execute(w, struct {
		GrantID   string
		UserCode  string
		Requested []gnap.AccessItem
	}{
		GrantID:   g.ID,
		UserCode:  deref(g.UserCode),
		Requested: g.RequestedAccess, // shows type/resource/actions/constraints/locations
	})
}
