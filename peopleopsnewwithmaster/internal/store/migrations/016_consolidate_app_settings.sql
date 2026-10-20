-- Consolidate global application settings into one key/value table.
CREATE TABLE IF NOT EXISTS dcs_app_settings (
  setting_key TEXT PRIMARY KEY,
  setting_value TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by TEXT NULL REFERENCES users(id) ON DELETE SET NULL
);
INSERT INTO dcs_app_settings(setting_key,setting_value) VALUES
 ('currency','GBP'),
 ('timezone','Asia/Dubai')
ON CONFLICT(setting_key) DO NOTHING;
DO $$ BEGIN
 IF to_regclass('public.dcs_app_currency') IS NOT NULL THEN
   INSERT INTO dcs_app_settings(setting_key,setting_value)
   SELECT 'currency', btrim(currency_code) FROM dcs_app_currency WHERE is_default=true LIMIT 1
   ON CONFLICT(setting_key) DO UPDATE SET setting_value=EXCLUDED.setting_value;
 END IF;
 IF to_regclass('public.dcs_app_timezone') IS NOT NULL THEN
   INSERT INTO dcs_app_settings(setting_key,setting_value)
   SELECT 'timezone', CASE WHEN timezone_name='Arabian Standard Time' THEN 'Asia/Dubai' ELSE timezone_name END FROM dcs_app_timezone WHERE active=true AND is_default=true LIMIT 1
   ON CONFLICT(setting_key) DO UPDATE SET setting_value=EXCLUDED.setting_value;
 END IF;
END $$;
DROP TABLE IF EXISTS dcs_app_currency;
DROP TABLE IF EXISTS dcs_app_timezone;
