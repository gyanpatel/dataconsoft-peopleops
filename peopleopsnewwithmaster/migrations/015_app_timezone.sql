-- Central application time zone and currency configuration.
CREATE TABLE IF NOT EXISTS dcs_app_timezone (
  timezone_id BIGSERIAL PRIMARY KEY,
  timezone_name TEXT NOT NULL UNIQUE,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO dcs_app_timezone(timezone_name,active,is_default)
VALUES ('Arabian Standard Time',TRUE,TRUE)
ON CONFLICT(timezone_name) DO UPDATE SET active=TRUE,is_default=TRUE,updated_at=now();
CREATE UNIQUE INDEX IF NOT EXISTS idx_dcs_app_timezone_single_default ON dcs_app_timezone(is_default) WHERE is_default=TRUE;
delete from dcs_app_currency ;
INSERT INTO dcs_app_currency(currency_code,currency_name,active,is_default) VALUES
 ('GBP','Pound sterling',TRUE,TRUE),('AED','UAE dirham',TRUE,FALSE),('USD','US dollar',TRUE,FALSE),('EUR','Euro',TRUE,FALSE)
ON CONFLICT(currency_code) DO NOTHING;
