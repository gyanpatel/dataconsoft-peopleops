package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"strings"

	"peopleops/internal/auth"

	"github.com/jackc/pgx/v5"
)

const signingPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Review and sign employment document</title>
<style>
:root{font-family:system-ui,-apple-system,sans-serif;color:#172033;background:#f4f6f8}body{max-width:960px;margin:24px auto;padding:16px}h1{font-size:1.65rem}.card{background:#fff;border:1px solid #dce2e8;border-radius:14px;padding:20px;margin:16px 0}.preview{width:100%;height:65vh;border:1px solid #cbd5e1;border-radius:8px}.signature-pad{display:block;width:100%;height:180px;border:1px solid #9aa7b7;border-radius:8px;touch-action:none;background:#fff}.row{display:flex;gap:12px;flex-wrap:wrap;align-items:center}input[type=text]{padding:10px;border:1px solid #aab4c2;border-radius:7px;min-width:240px;max-width:100%}button{padding:11px 17px;border:0;border-radius:8px;background:#176b50;color:white;font-weight:650;cursor:pointer}button.secondary{background:#e8edf2;color:#172033}button:disabled{opacity:.55;cursor:wait}.muted{color:#536174;font-size:.92rem}.error{color:#b42318}.sig-preview{font-family:cursive;font-style:italic;font-size:1.7rem}.consent{display:flex;gap:10px;align-items:flex-start;margin:16px 0}
</style></head><body>
<h1>Review and sign your employment document</h1><p class="muted">Read the complete document before signing. If any details are incorrect, contact HR and do not sign until corrected.</p>
<section class="card"><h2 id="docname">Employment document</h2><iframe class="preview" id="preview" title="Employment document preview"></iframe><p class="muted">The preview is the document currently held by PeopleOps. Your signature will be added to the final page in the designated signature area configured by HR.</p></section>
<section class="card"><h2>Your signature</h2><p>Type your name for a script-style signature, or draw your signature in the box.</p><div class="row"><label for="typed">Type full name</label><input id="typed" type="text" maxlength="120" autocomplete="name" placeholder="Full legal name"><span id="typedPreview" class="sig-preview"></span></div><p class="muted">Or draw below using a mouse, touch screen, or stylus.</p><canvas id="pad" class="signature-pad" aria-label="Draw your signature"></canvas><div class="row" style="margin-top:10px"><button class="secondary" id="clear" type="button">Clear drawing</button><label><input id="useDrawing" type="checkbox"> Use drawn signature instead of typed name</label></div>
<label class="consent"><input id="consent" type="checkbox"><span>I have reviewed this document, intend to sign it electronically, and agree that my electronic signature may be used as my signature on this document.</span></label><button id="sign">Sign document</button><p id="message" role="status" class="muted"></p></section>
<script src="https://unpkg.com/pdf-lib@1.17.1/dist/pdf-lib.min.js"></script><script>
(()=>{const token=location.pathname.split('/').filter(Boolean).pop();const preview=document.getElementById('preview');preview.src=location.pathname+'/preview';const canvas=document.getElementById('pad'),ctx=canvas.getContext('2d'),typed=document.getElementById('typed'),typedPreview=document.getElementById('typedPreview'),message=document.getElementById('message'),sign=document.getElementById('sign');let drawing=false,hasDrawing=false;
function sizeCanvas(){const dpr=window.devicePixelRatio||1,rect=canvas.getBoundingClientRect();canvas.width=Math.max(1,rect.width*dpr);canvas.height=Math.max(1,rect.height*dpr);ctx.setTransform(dpr,0,0,dpr,0,0);ctx.lineWidth=2.2;ctx.lineCap='round';ctx.lineJoin='round';ctx.strokeStyle='#172033'}sizeCanvas();window.addEventListener('resize',sizeCanvas);
function point(e){const r=canvas.getBoundingClientRect();return{x:e.clientX-r.left,y:e.clientY-r.top}}canvas.addEventListener('pointerdown',e=>{drawing=true;hasDrawing=true;canvas.setPointerCapture(e.pointerId);const p=point(e);ctx.beginPath();ctx.moveTo(p.x,p.y)});canvas.addEventListener('pointermove',e=>{if(!drawing)return;const p=point(e);ctx.lineTo(p.x,p.y);ctx.stroke()});for(const ev of ['pointerup','pointercancel','pointerleave'])canvas.addEventListener(ev,()=>drawing=false);document.getElementById('clear').onclick=()=>{ctx.clearRect(0,0,canvas.width,canvas.height);hasDrawing=false};typed.addEventListener('input',()=>typedPreview.textContent=typed.value);
function signaturePNG(){if(document.getElementById('useDrawing').checked){if(!hasDrawing)throw Error('Please draw your signature first.');return canvas.toDataURL('image/png')}const name=typed.value.trim();if(!name)throw Error('Enter your full name or draw your signature.');const c=document.createElement('canvas');c.width=1200;c.height=240;const x=c.getContext('2d');x.clearRect(0,0,c.width,c.height);x.fillStyle='#172033';x.font='italic 84px cursive';x.fillText(name,24,150);return c.toDataURL('image/png')}
function dataBytes(data){const raw=atob(data.split(',')[1]),out=new Uint8Array(raw.length);for(let i=0;i<raw.length;i++)out[i]=raw.charCodeAt(i);return out}
sign.onclick=async()=>{try{if(!document.getElementById('consent').checked)throw Error('Please confirm your consent to sign electronically.');if(!window.PDFLib)throw Error('The signing component could not load. Please retry or contact HR.');sign.disabled=true;message.textContent='Preparing signed document…';const pdfResponse=await fetch(location.pathname+'/preview',{cache:'no-store'});if(!pdfResponse.ok)throw Error('Could not load the document preview.');const pdfBytes=await pdfResponse.arrayBuffer();const pdf=await PDFLib.PDFDocument.load(pdfBytes);const pages=pdf.getPages();if(!pages.length)throw Error('The document has no pages.');const page=pages[pages.length-1],{width,height}=page.getSize(),png=await pdf.embedPng(dataBytes(signaturePNG()));
// Default signature block is configurable via SIGNATURE_BLOCK_PAGE / SIGNATURE_BLOCK_X / SIGNATURE_BLOCK_Y in server-side template tooling; until configured, place at the lower-right of the final page.
const w=Math.min(220,width*.34),h=w*png.height/png.width,x=Math.max(24,width-w-48),y=42;page.drawImage(png,{x,y,width:w,height:Math.min(h,70)});page.drawText('Electronically signed by '+(typed.value.trim()||'Employee'),{x,y:Math.max(20,y-16),size:8,color:PDFLib.rgb(.15,.18,.23)});page.drawText('Signed: '+new Date().toISOString(),{x,y:Math.max(10,y-27),size:7,color:PDFLib.rgb(.3,.34,.4)});const signed=await pdf.save();let binary='';for(let i=0;i<signed.length;i+=0x8000)binary+=String.fromCharCode(...signed.subarray(i,Math.min(i+0x8000,signed.length)));const base64=btoa(binary);const resp=await fetch(location.pathname,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({signedPdf:base64,signerName:typed.value.trim(),signatureMethod:document.getElementById('useDrawing').checked?'drawn':'typed',consent:true})});const result=await resp.json().catch(()=>({}));if(!resp.ok)throw Error(result.error||'Signing failed. Please contact HR.');document.querySelector('body').innerHTML='<main class="card"><h1>Document signed</h1><p>Your electronically signed document has been saved. The document status is now Digitally signed.</p><p>You may close this page.</p></main>';}catch(e){message.className='error';message.textContent=e.message||'Could not sign document.';sign.disabled=false}};
})();</script></body></html>`

func signingToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

// createEmploymentContract currently creates the app's configured draft summary. A sanitized, approved
// contract template must be configured before this is suitable for issuing binding employment terms.
func (s *Server) createEmploymentContract(employeeID string) error {
	var first, last, email, department, title, location, start, currency string
	var gross int64
	err := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT e.first_name,e.last_name,e.email,e.department,e.job_title,e.location,e.start_date::text,COALESCE(b.gross_pay_minor,e.salary_minor),COALESCE(b.currency,e.currency,'GBP') FROM employees e LEFT JOIN dsc_employee_salary_breakup b ON b.employee_id=e.id WHERE e.id=$1`, employeeID).Scan(&first, &last, &email, &department, &title, &location, &start, &gross, &currency)
	if err != nil {
		return err
	}
	// Retry email delivery for an existing pending contract. Previously, a failed
	// SMTP send left a pending row behind, and later calls returned without emailing.
	var existingID, existingStatus string
	lookupErr := s.Store.DB.QueryRow(s.Store.Ctx, `SELECT id, COALESCE(signing_status,'') FROM dcs_doc_store WHERE employee_id=$1 AND document_type='Contract' AND signing_status IN ('Pending signature','Digitally signed','Signed') ORDER BY uploaded_at DESC LIMIT 1`, employeeID).Scan(&existingID, &existingStatus)
	if lookupErr == nil && (existingStatus == "Digitally signed" || existingStatus == "Signed") {
		return nil
	}
	token, hash, err := signingToken()
	if err != nil {
		return err
	}
	if lookupErr == nil && existingStatus == "Pending signature" {
		if _, err = s.Store.DB.Exec(s.Store.Ctx, `UPDATE dcs_doc_store SET signing_token_hash=$2, updated_at=now() WHERE id=$1 AND signing_status='Pending signature'`, existingID, hash); err != nil {
			return err
		}
		return sendSigningRequest(email, first, token)
	}
	if lookupErr != nil && lookupErr != pgx.ErrNoRows {
		return lookupErr
	}
	content := fmt.Sprintf("DRAFT EMPLOYMENT TERMS — FOR REVIEW AND SIGNATURE\n\nEmployee: %s %s\nEmail: %s\nDepartment: %s\nPosition: %s\nWork location: %s\nEmployment start date: %s\nGross annual salary: %s %.2f\n\nThis automatically generated document is a draft summary, not a substitute for your organisation's approved employment contract. HR must verify and attach the approved contractual terms before relying on it.\n\nEmployee signature: __________________________________\nDate: _______________________________________________\n", first, last, email, department, title, location, start, currency, float64(gross)/100)
	pdf := simplePDF(content)
	id := auth.NewID()
	name := fmt.Sprintf("Employment contract - %s %s - draft.pdf", first, last)
	_, err = s.Store.DB.Exec(s.Store.Ctx, `INSERT INTO dcs_doc_store(id,employee_id,document_name,document_type,mime_type,file_size,doc_file,verification_status,notes,signing_status,signing_token_hash) VALUES($1,$2,$3,'Contract','application/pdf',$4,$5,'Pending','Automatically generated draft; HR review required before use','Pending signature',$6)`, id, employeeID, name, len(pdf), pdf, hash)
	if err != nil {
		return err
	}
	if err = sendSigningRequest(email, first, token); err != nil {
		return err
	}
	return nil
}

func sendSigningRequest(to, first, token string) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	user := os.Getenv("SMTP_USERNAME")
	pass := os.Getenv("SMTP_PASSWORD")
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if host == "" || from == "" {
		return fmt.Errorf("SMTP_HOST and SMTP_FROM must be configured")
	}
	if port == "" {
		port = "587"
	}
	origin := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_APP_URL")), "/")
	if origin == "" {
		origin = strings.TrimRight(strings.TrimSpace(os.Getenv("APP_ORIGIN")), "/")
	}
	if origin == "" {
		origin = "https://peopleops.dataconsoft.com"
	}
	link := origin + "/sign/" + token
	body := "To: " + to + "\r\nSubject: Employment document ready for signature\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nHello " + first + ",\r\n\r\nAn employment document is ready for your review and electronic signature: " + link + "\r\n\r\nPlease review the document carefully. Contact HR if the terms are incomplete or incorrect.\r\n"
	var a smtp.Auth
	if user != "" {
		a = smtp.PlainAuth("", user, pass, host)
	}
	return smtp.SendMail(host+":"+port, a, from, []string{to}, []byte(body))
}

func (s *Server) signingDocument(r *http.Request) (string, string, []byte, error) {
	token := r.PathValue("token")
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	var id, name string
	var data []byte
	err := s.Store.DB.QueryRow(r.Context(), `SELECT id,document_name,doc_file FROM dcs_doc_store WHERE signing_token_hash=$1 AND signing_status='Pending signature'`, hash).Scan(&id, &name, &data)
	return id, name, data, err
}

func (s *Server) previewEmploymentDocument(w http.ResponseWriter, r *http.Request) {
	_, name, data, err := s.signingDocument(r)
	if err != nil {
		http.Error(w, "This signing link is invalid, expired, or already used.", http.StatusGone)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", name))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) signEmploymentDocument(w http.ResponseWriter, r *http.Request) {
	id, name, _, err := s.signingDocument(r)
	if err != nil {
		if r.Method == http.MethodGet {
			http.Error(w, "This signing link is invalid, expired, or already used.", http.StatusGone)
		} else {
			http.Error(w, `{"error":"This signing link is invalid, expired, or already used."}`, http.StatusGone)
		}
		return
	}
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		page := strings.Replace(signingPage, "<h2 id=\"docname\">Employment document</h2>", "<h2 id=\"docname\">"+html.EscapeString(name)+"</h2>", 1)
		_, _ = io.WriteString(w, page)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	writeJSONError := func(status int, msg string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentBytes*2)
	var in struct {
		SignedPDF       string `json:"signedPdf"`
		SignerName      string `json:"signerName"`
		SignatureMethod string `json:"signatureMethod"`
		Consent         bool   `json:"consent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONError(http.StatusBadRequest, "Invalid signing request.")
		return
	}
	if !in.Consent || strings.TrimSpace(in.SignerName) == "" || len(in.SignerName) > 120 || (in.SignatureMethod != "typed" && in.SignatureMethod != "drawn") {
		writeJSONError(http.StatusBadRequest, "Provide your name, signature method, and electronic-signature consent.")
		return
	}
	data, err := base64.StdEncoding.DecodeString(in.SignedPDF)
	if err != nil || len(data) < 5 || len(data) > maxDocumentBytes || !strings.HasPrefix(string(data[:5]), "%PDF-") {
		writeJSONError(http.StatusBadRequest, "The signed PDF is invalid or exceeds the 15 MB limit.")
		return
	}
	tag, err := s.Store.DB.Exec(r.Context(), `UPDATE dcs_doc_store SET doc_file=$2,file_size=$3,mime_type='application/pdf',document_name=$4,verification_status='Pending',signing_status='Digitally signed',signed_at=now(),signing_token_hash=NULL,notes=COALESCE(notes,'') || $5,updated_at=now() WHERE id=$1 AND signing_token_hash=$6 AND signing_status='Pending signature'`, id, data, len(data), strings.TrimSuffix(name, "- draft.pdf")+"- signed.pdf", "\nElectronic signature recorded: "+strings.TrimSpace(in.SignerName)+" ("+in.SignatureMethod+"); consent given at submission.", tokenHash(r.PathValue("token")))
	if err != nil || tag.RowsAffected() != 1 {
		writeJSONError(http.StatusInternalServerError, "Could not save signed document; contact HR.")
		return
	}
	_, _ = s.Store.DB.Exec(r.Context(), `INSERT INTO dcs_document_signing_audit(document_id,signer_name,signature_method,consent_text,signing_ip,user_agent,signed_at) VALUES($1,$2,$3,$4,$5,$6,now())`, id, strings.TrimSpace(in.SignerName), in.SignatureMethod, "I have reviewed this document, intend to sign it electronically, and agree that my electronic signature may be used as my signature on this document.", clientIP(r), r.UserAgent())
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "Digitally signed"})
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return strings.Trim(host[:i], "[]")
	}
	return host
}
