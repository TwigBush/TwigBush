// ---- Sample grant body -------------------------------------------------------
const sample = {
    client: { key:{ proof:"httpsig", jwk:{ kty:"EC", crv:"P-256", x:"X", y:"Y" } } },
    access: [{
        type:"payment", resource_id:"sku:GPU-HOURS-100", actions:["purchase"],
        constraints:{ amount:"19.99", currency:"USD", merchant_id:"merchant:acme" },
        locations:["https://localhost:4000/"]
    }],
    interact:{ start:["user_code"] },
    token_format:"jwt"
};

// ---- State -------------------------------------------------------------------
let currentGrantId = null;
let continueUri = null;
let contToken = null;
let pollTimer = null;

// ---- Helpers -----------------------------------------------------------------
function $(id){return document.getElementById(id);}
function setGrantText(){ $('grantJSON').value = JSON.stringify(sample, null, 2); }

function stateBadge(state){
    const s = (state||'').toLowerCase();
    const cls = ['processing','pending','approved','denied','expired','finalized'].includes(s) ? s : 'pending';
    $('stateBadge').innerHTML = `<span class="dot ${cls}"></span><span>${s || '—'}</span>`;
}

function diagramDef(state="pending"){
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
  note right of ${state}: current
`;
}

async function setDiagram(state="pending"){
    const def = diagramDef(state);
    try {
        const { svg } = await mermaid.render("grantDiagram", def);
        $('diagram').innerHTML = svg;
    } catch (e) {
        $('diagram').textContent = def;
        console.error("Mermaid render failed:", e);
    }
}

function addEventLine(kind, details=""){
    const li = document.createElement('li');
    const t = new Date().toLocaleTimeString();
    let text = "";
    switch(kind){
        case 'grant_created': text = `📝 Grant created ${details}`; break;
        case 'code_issued':   text = `🔐 User code issued ${details}`; break;
        case 'code_verified': text = `✅ Code verified ${details}`; break;
        case 'approved':      text = `✅ Approved ${details}`; break;
        case 'denied':        text = `⛔ Denied ${details}`; break;
        case 'expired':       text = `⌛ Expired ${details}`; break;
        case 'finalized':     text = `🎟️ Token issued ${details}`; break;
        case 'continue':      text = `➡️ Continue called ${details}`; break;
        default:              text = `${details}`;
    }
    li.textContent = `[${t}] ${text}`;
    $('events').prepend(li);
}

async function postJSON(url, body){
    const res = await fetch(url, { method:"POST",
        headers:{ "Content-Type":"application/json" },
        body: JSON.stringify(body)
    });
    if(!res.ok) throw new Error(await res.text());
    return res.json();
}

async function postForm(url, data){
    const res = await fetch(url, { method:"POST",
        headers:{ "Content-Type":"application/x-www-form-urlencoded" },
        body: new URLSearchParams(data)
    });
    if(!res.ok) throw new Error(await res.text());
    return res.text();
}

async function authGetJSON(url, token){
    const res = await fetch(url, { method: 'POST', headers:{ "Authorization": `GNAP ${token}` }});
    if(!res.ok) throw new Error(await res.text());
    return res.json();
}

// ---- Device code UX helpers --------------------------------------------------
function normalizeCode(v){
    v = (v||'').toUpperCase().replace(/[^A-Z0-9]/g,'');
    if(v.length>4) v = v.slice(0,4) + '-' + v.slice(4,8);
    return v.slice(0,9);
}
function setIssuedCode(code){
    $('issuedCode').textContent = code || '—';
    $('inputCode').value = code || '';
}

function preview(str, head=12, tail=8) {
    if (!str || typeof str !== 'string') return '—';
    if (str.length <= head + tail + 1) return str;
    return str.slice(0, head) + '…' + str.slice(-tail);
}

// ---- SSE ---------------------------------------------------------------------
function startSSE(){
    const es = new EventSource("/events");
    es.addEventListener("grant", (e) => {
        try {
            const msg = JSON.parse(e.data);
            // msg.id, msg.state, msg.updated_at
            if(msg.id === currentGrantId){
                stateBadge(msg.state);
                setDiagram(msg.state);
            }
            const dpretty = `id=${msg.id} → ${msg.state}`;
            addEventLine(msg.state?.toLowerCase(), dpretty);
        } catch (err) {
            console.error("bad SSE grant event", err);
        }
    });
    es.addEventListener("ping", () => {});
}

// ---- Continue helpers --------------------------------------------------------
async function callContinue(){
    if(!currentGrantId || !contToken || !continueUri){
        $('continueOut').textContent = "No continue data yet. Create a grant first.";
        return;
    }
    try{
        const data = await authGetJSON(continueUri, contToken);
        $('continueOut').textContent = JSON.stringify(data, null, 2);
        addEventLine('continue', `grant=${currentGrantId}`);
        // If access_token is returned, mark finalized
        if(data.access_token){
            stateBadge('finalized');
            setDiagram('finalized');
            addEventLine('finalized', `token issued`);
            const tok = data?.access_token?.value;
            if (tok) {
                $('curContTok').textContent = preview(tok, 18, 12);
            }
        }
    }catch(e){
        $('continueOut').textContent = `Error: ${e.message}`;
    }
}

function startPolling(){
    if(pollTimer) return;
    pollTimer = setInterval(callContinue, 2000);
}
function stopPolling(){
    if(pollTimer) clearInterval(pollTimer);
    pollTimer = null;
}

// ---- Init --------------------------------------------------------------------
window.addEventListener('DOMContentLoaded', () => {
    setGrantText();
    setDiagram("processing");
    stateBadge('processing');
    startSSE();

    // Create grant
    $('btnGrant').onclick = async () => {
        try {
            const body = JSON.parse($('grantJSON').value || "{}");
            const resp = await postJSON("/grant", body);

            // Extract grant id, continue URI/token, user code
            const m = /\/continue\/([^/]+)$/.exec(resp?.continue?.uri || "");
            currentGrantId = m ? m[1] : null;
            continueUri    = resp?.continue?.uri || null;
            contToken      = resp?.continue?.access_token || null;

            $('curGrant').textContent    = currentGrantId || '—';
            $('curContinue').textContent = preview(continueUri, 28, 18);
            $('curContTok').textContent  = preview(contToken, 18, 12);

            const code = resp?.interact?.user_code?.code || '';
            setIssuedCode(code);

            addEventLine('grant_created', `grant=${currentGrantId}`);
            if(code) addEventLine('code_issued', `code=${code}`);

            stateBadge('pending');
            setDiagram('pending');
        } catch (e) {
            addEventLine('error', "grant error: " + e.message);
            console.error(e);
        }
    };

    // Debug approve/deny (requires demo endpoints)
    $('btnApprove').onclick = async () => {
        try {
            if (!currentGrantId) return;
            await postForm(`/debug/approve/${currentGrantId}`, {});
            addEventLine('approved', `grant=${currentGrantId}`);
        } catch (e) {
            addEventLine('error', "approve error: " + e.message);
        }
    };
    $('btnDeny').onclick = async () => {
        try {
            if (!currentGrantId) return;
            await postForm(`/debug/deny/${currentGrantId}`, {});
            addEventLine('denied', `grant=${currentGrantId}`);
        } catch (e) {
            addEventLine('error', "deny error: " + e.message);
        }
    };

    // Simulate code verify (form POST to /device/verify)
    $('inputCode').addEventListener('input', (e) => {
        const v = normalizeCode(e.target.value);
        if(v !== e.target.value){
            e.target.value = v;
            e.target.setSelectionRange(v.length, v.length);
        }
    });
    $('btnVerifyCode').onclick = async () => {
        const code = $('inputCode').value.trim();
        if(!/^[A-Z0-9]{4}-[A-Z0-9]{4}$/.test(code)){
            $('verifyMsg').textContent = "Invalid format. Use ABCD-1234.";
            return;
        }
        try{
            await postForm("/device/verify", { user_code: code });
            $('verifyMsg').textContent = "Code accepted. Review consent in the other tab or Approve/Deny here.";
            addEventLine('code_verified', `code=${code}`);
            $('btnApprove').disabled = false;   // <-- unlock approve
        }catch(e){
            $('verifyMsg').textContent = "Verification failed: " + e.message;
        }
    };

    // Continue flow
    $('btnContinue').onclick = callContinue;
    $('btnPoll').onclick = startPolling;
    $('btnStopPoll').onclick = stopPolling;
});
