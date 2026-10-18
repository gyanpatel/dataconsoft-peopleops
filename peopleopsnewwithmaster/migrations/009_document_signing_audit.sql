CREATE TABLE IF NOT EXISTS dcs_document_signing_audit (
  id BIGSERIAL PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES dcs_doc_store(id) ON DELETE CASCADE,
  signer_name TEXT NOT NULL,
  signature_method TEXT NOT NULL CHECK (signature_method IN ('typed','drawn')),
  consent_text TEXT NOT NULL,
  signing_ip TEXT,
  user_agent TEXT,
  signed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_dcs_document_signing_audit_document_id
  ON dcs_document_signing_audit(document_id, signed_at DESC);
