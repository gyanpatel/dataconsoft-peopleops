CREATE TABLE IF NOT EXISTS dcs_app_settings (
  setting_key TEXT PRIMARY KEY,
  setting_value TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_by TEXT REFERENCES users(id)
);
INSERT INTO dcs_app_settings(setting_key, setting_value)
VALUES ('document_expiry_notice_days', '30')
ON CONFLICT (setting_key) DO NOTHING;
