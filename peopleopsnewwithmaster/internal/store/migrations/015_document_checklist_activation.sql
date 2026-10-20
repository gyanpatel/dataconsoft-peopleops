-- Checklist document types gate employee activation until every required type is verified.
ALTER TABLE dcs_document_types ADD COLUMN IF NOT EXISTS check_list BOOLEAN NOT NULL DEFAULT FALSE;

INSERT INTO dcs_document_types(name, active, check_list) VALUES
  ('Passport', TRUE, TRUE),
  ('Emirates ID', TRUE, TRUE),
  ('IBAN letter from bank', TRUE, TRUE)
ON CONFLICT (lower(name)) DO UPDATE SET active=TRUE, check_list=TRUE;

-- Ensure the remaining common document categories are available without marking them as onboarding requirements.
INSERT INTO dcs_document_types(name, active, check_list) VALUES
  ('Identity', TRUE, FALSE),
  ('Contract', TRUE, FALSE),
  ('Right to work', TRUE, FALSE),
  ('Qualification', TRUE, FALSE),
  ('Other', TRUE, FALSE)
ON CONFLICT (lower(name)) DO NOTHING;
