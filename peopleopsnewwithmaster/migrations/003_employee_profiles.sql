ALTER TABLE employees ADD COLUMN IF NOT EXISTS photo_document_id TEXT REFERENCES dcs_doc_store(id) ON DELETE SET NULL;

