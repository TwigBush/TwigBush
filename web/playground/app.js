// ==============================
// GNAP Playground - Full Script
// ==============================

// ---- Sample grant body -------------------------------------------------------
const sample = {
    access_token: [
        {
            label: "chucks",
            access: [
                {
                    type: "shop.purchase",
                    actions: [
                        "purchase"
                    ],
                    identifier: "intent-chucks-1759345834468-ba606bc35980d",
                    locations: [
                        "https://api.chucks.example/checkout"
                    ],
                    constraints: {
                        profile: "plan_purchase",
                        account_id: "acct1",
                        merchant_id: "chucks",
                        line_items: [
                            {
                                sku: "chicken_breast_1lb",
                                qty: 1
                            },
                            {
                                sku: "eggs_1dozen",
                                qty: 1
                            },
                            {
                                sku: "garlic_head",
                                qty: 1
                            },
                            {
                                sku: "red_pepper_flakes",
                                qty: 1
                            },
                            {
                                sku: "salt",
                                qty: 1
                            }
                        ],
                        currency: "USD",
                        amount_cents: 7416,
                        servings: 4,
                        plan_id: "chicken_noodle_soup",
                        quote_expires_at: "2025-10-01T19:20:32Z",
                        domain: "food",
                        schedule: {
                            kind: "asap"
                        }
                    }
                }
            ]
        },
        {
            label: "wallystore",
            access: [
                {
                    type: "shop.purchase",
                    actions: [
                        "purchase"
                    ],
                    identifier: "intent-wallystore-1759345834468-f0d7f9da8811e",
                    locations: [
                        "https://api.wallystore.example/checkout"
                    ],
                    constraints: {
                        profile: "plan_purchase",
                        account_id: "acct1",
                        merchant_id: "wallystore",
                        line_items: [
                            {
                                sku: "black_pepper",
                                qty: 1
                            },
                            {
                                sku: "olive_oil",
                                qty: 1
                            },
                            {
                                sku: "rice_2lb",
                                qty: 1
                            },
                            {
                                sku: "tomato_diced_14oz",
                                qty: 1
                            }
                        ],
                        currency: "USD",
                        amount_cents: 9336,
                        servings: 4,
                        plan_id: "chicken_noodle_soup",
                        quote_expires_at: "2025-10-01T19:20:33Z",
                    }
                }
            ]
        }
    ],
    client: {
        key: {
            proof: "httpsig",
            jwk: {
                kid: "ec1",
                kty: "EC",
                crv: "P-256",
                x: "abc",
                y: "def",
                use: "sig",
                alg: "ES256"
            }
        }
    },
    interact: {
        start: [
            "user_code"
        ]
    }
}


// ---- Bases -------------------------------------------------------------------
// Sources: querystring ?as=... ?ui=..., then window globals, then localStorage, then defaults
const AS_BASE =
    new URLSearchParams(location.search).get("as") ||
    (typeof window !== "undefined" && window.__AS_BASE__) ||
    localStorage.getItem("AS_BASE") ||
    "http://localhost:8085";

const PLAYGROUND_BASE =
    new URLSearchParams(location.search).get("ui") ||
    (typeof window !== "undefined" && window.__PLAYGROUND_BASE__) ||
    localStorage.getItem("PLAYGROUND_BASE") ||
    ""; // same origin default

// Persist if provided via querystring
(() => {
    const qs = new URLSearchParams(location.search);
    if (qs.get("as")) localStorage.setItem("AS_BASE", AS_BASE);
    if (qs.get("ui")) localStorage.setItem("PLAYGROUND_BASE", PLAYGROUND_BASE);
})();

function joinURL(base, path) {
    if (!base) return path;
    if (/^https?:\/\//i.test(path)) return path;
    const left = base.endsWith("/") ? base.slice(0, -1) : base;
    const right = path.startsWith("/") ? path : `/${path}`;
    return `${left}${right}`;
}

// ---- State -------------------------------------------------------------------
let currentGrantId = null;
let continueUri = null;
let contToken = null;
let pollTimer = null;

// ---- Helpers -----------------------------------------------------------------
function $(id) { return document.getElementById(id); }
function setGrantText() { $("grantJSON").value = JSON.stringify(sample, null, 2); }

function stateBadge(state) {
    const s = (state || "").toLowerCase();
    const valid = ["processing", "pending", "approved", "denied", "expired", "finalized"];
    const cls = valid.includes(s) ? s : "pending";
    $("stateBadge").innerHTML = `<span class="dot ${cls}"></span><span>${s || "—"}</span>`;
}

function diagramDef() {
    return `
stateDiagram-v2
  [*] --> processing: create
  processing --> pending: interact/user_code
  pending --> approved: user approves
  pending --> denied: user denies
  pending --> expired: ttl
  approved --> finalized: continue/issue token
  denied --> finalized
  expired --> finalized
`;
}

async function setDiagram() {
    const def = diagramDef();
    try {
        const { svg } = await mermaid.render("grantDiagram", def);
        $("diagram").innerHTML = svg;
    } catch (e) {
        $("diagram").textContent = def;
        console.error("Mermaid render failed:", e);
    }
}

function addEventLine(kind, details = "") {
    const li = document.createElement("li");
    const t = new Date().toLocaleTimeString();
    let text = "";
    switch ((kind || "").toLowerCase()) {
        case "grant_created": text = `📝 Grant created ${details}`; break;
        case "code_issued":   text = `🔐 User code issued ${details}`; break;
        case "code_verified": text = `✅ Code verified ${details}`; break;
        case "approved":      text = `✅ Approved ${details}`; break;
        case "denied":        text = `⛔ Denied ${details}`; break;
        case "expired":       text = `⌛ Expired ${details}`; break;
        case "finalized":     text = `🎟️ Token issued ${details}`; break;
        case "continue":      text = `➡️ Continue called ${details}`; break;
        default:              text = `${details}`;
    }
    li.textContent = `[${t}] ${text}`;
    $("events").prepend(li);
}

async function postJSON(url, body) {
    const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body)
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function postForm(url, data) {
    const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams(data)
    });
    if (!res.ok) throw new Error(await res.text());
    return res.text();
}

async function authGetJSON(url, token) {
    const res = await fetch(url, {
        method: "POST",
        headers: { "Authorization": `GNAP ${token}` }
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

// ---- Device code UX helpers --------------------------------------------------
function normalizeCode(v) {
    v = (v || "").toUpperCase().replace(/[^A-Z0-9]/g, "");
    if (v.length > 4) v = v.slice(0, 4) + "-" + v.slice(4, 8);
    return v.slice(0, 9);
}
function setIssuedCode(code) {
    $("issuedCode").textContent = code || "—";
    $("inputCode").value = code || "";
}

function preview(str, head = 12, tail = 8) {
    if (!str || typeof str !== "string") return "—";
    if (str.length <= head + tail + 1) return str;
    return str.slice(0, head) + "…" + str.slice(-tail);
}

// ---- SSE ---------------------------------------------------------------------
function startSSE() {
    const es = new EventSource(joinURL(PLAYGROUND_BASE, "/events"));
    es.addEventListener("grant", (e) => {
        try {
            const msg = JSON.parse(e.data);
            if (msg.id === currentGrantId) {
                stateBadge(msg.state);
                setDiagram();
            }
            const dpretty = `id=${msg.id} → ${msg.state}`;
            addEventLine((msg.state || "").toLowerCase(), dpretty);
        } catch (err) {
            console.error("bad SSE grant event", err);
        }
    });
    es.addEventListener("ping", () => {});
}

// ---- Continue helpers --------------------------------------------------------
async function callContinue() {
    if (!currentGrantId || !contToken || !continueUri) {
        $("continueOut").textContent = "No continue data yet. Create a grant first.";
        return;
    }
    try {
        const data = await authGetJSON(continueUri, contToken);
        $("continueOut").textContent = JSON.stringify(data, null, 2);
        addEventLine("continue", `grant=${currentGrantId}`);

        if (data.access_token) {
            stateBadge("finalized");
            setDiagram();
            addEventLine("finalized", "token issued");
            const tok = data?.access_token?.value;
            if (tok) $("curContTok").textContent = preview(tok, 18, 12);
        }
    } catch (e) {
        $("continueOut").textContent = `Error: ${e.message}`;
    }
}

function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(callContinue, 2000);
}
function stopPolling() {
    if (pollTimer) clearInterval(pollTimer);
    pollTimer = null;
}

// ---- Init --------------------------------------------------------------------
window.addEventListener("DOMContentLoaded", () => {
    setGrantText();
    setDiagram();
    stateBadge("processing");
    startSSE();

    // Reflect the AS device page link
    const devLink = $("deviceLink");
    if (devLink) devLink.href = joinURL(AS_BASE, "/device");

    // Create grant
    $("btnGrant").onclick = async () => {
        try {
            const body = JSON.parse($("grantJSON").value || "{}");
            // GNAP: POST /grants on the AS
            const resp = await postJSON(joinURL(AS_BASE, "/grants"), body);

            // Resolve continue URI absolute
            const contUriRaw = resp?.continue?.uri || "";
            const absoluteContinue = joinURL(AS_BASE, contUriRaw);
            const m = /\/continue\/([^/]+)$/.exec(absoluteContinue);
            currentGrantId = m ? m[1] : null;
            continueUri = absoluteContinue || null;
            contToken = resp?.continue?.access_token || null;

            $("curGrant").textContent = currentGrantId || "—";
            $("curContinue").textContent = preview(continueUri, 28, 18);
            $("curContTok").textContent = preview(contToken, 18, 12);

            const code = resp?.interact?.user_code?.code || "";
            setIssuedCode(code);

            addEventLine("grant_created", `grant=${currentGrantId}`);
            if (code) addEventLine("code_issued", `code=${code}`);

            stateBadge("pending");
            setDiagram();
        } catch (e) {
            addEventLine("error", "grant error: " + e.message);
            console.error(e);
        }
    };

    // Debug approve/deny (demo endpoints on playground)
    $("btnApprove").onclick = async () => {
        try {
            if (!currentGrantId) return;
            await postForm(joinURL(PLAYGROUND_BASE, `/debug/approve/${currentGrantId}`), {});
            addEventLine("approved", `grant=${currentGrantId}`);
            stateBadge("approved");
        } catch (e) {
            addEventLine("error", "approve error: " + e.message);
        }
    };

    $("btnDeny").onclick = async () => {
        try {
            if (!currentGrantId) return;
            await postForm(joinURL(PLAYGROUND_BASE, `/debug/deny/${currentGrantId}`), {});
            addEventLine("denied", `grant=${currentGrantId}`);
            stateBadge("denied");
        } catch (e) {
            addEventLine("error", "deny error: " + e.message);
        }
    };

    // Simulate code verify on the AS
    $("inputCode").addEventListener("input", (e) => {
        const v = normalizeCode(e.target.value);
        if (v !== e.target.value) {
            e.target.value = v;
            e.target.setSelectionRange(v.length, v.length);
        }
    });

    $("btnVerifyCode").onclick = async () => {
        const code = $("inputCode").value.trim();
        if (!/^[A-Z0-9]{4}-[A-Z0-9]{4}$/.test(code)) {
            $("verifyMsg").textContent = "Invalid format. Use ABCD-1234.";
            return;
        }
        try {
            await postJSON(joinURL(AS_BASE, "/device/verify"), { user_code: code });
            $("verifyMsg").textContent = "Code accepted. Review consent in the other tab or Approve or Deny here.";
            addEventLine("code_verified", `code=${code}`);
            $("btnApprove").disabled = false;
        } catch (e) {
            $("verifyMsg").textContent = "Verification failed: " + e.message;
        }
    };

    // Continue flow
    $("btnContinue").onclick = callContinue;
    $("btnPoll").onclick = startPolling;
    $("btnStopPoll").onclick = stopPolling;
});
