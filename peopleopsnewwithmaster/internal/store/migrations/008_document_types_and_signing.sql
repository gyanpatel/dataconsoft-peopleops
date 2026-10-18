CREATE TABLE IF NOT EXISTS dcs_document_types (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_dcs_document_types_lower_name ON dcs_document_types(lower(name));
INSERT INTO dcs_document_types(name) VALUES ('Identity'),('Contract'),('Right to work'),('Qualification'),('Other') ON CONFLICT DO NOTHING;
INSERT INTO dcs_document_types(name) SELECT DISTINCT trim(document_type) FROM dcs_doc_store WHERE trim(document_type)<>'' ON CONFLICT DO NOTHING;
ALTER TABLE dcs_doc_store ADD COLUMN IF NOT EXISTS signing_status TEXT NOT NULL DEFAULT 'Not required';
ALTER TABLE dcs_doc_store ADD COLUMN IF NOT EXISTS signing_token_hash TEXT;
ALTER TABLE dcs_doc_store ADD COLUMN IF NOT EXISTS signed_at TIMESTAMPTZ;
CREATE UNIQUE INDEX IF NOT EXISTS uq_dcs_doc_store_signing_token_hash ON dcs_doc_store(signing_token_hash) WHERE signing_token_hash IS NOT NULL;
