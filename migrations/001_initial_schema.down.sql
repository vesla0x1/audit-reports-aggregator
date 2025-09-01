DROP TRIGGER IF EXISTS update_audit_report_details_updated_at ON audit_report_details;
DROP TRIGGER IF EXISTS update_processes_updated_at ON processes;
DROP TRIGGER IF EXISTS update_downloads_updated_at ON downloads;
DROP TRIGGER IF EXISTS audit_report_provider_trigger ON audit_reports;
DROP TRIGGER IF EXISTS update_audit_reports_updated_at ON audit_reports;
DROP TRIGGER IF EXISTS update_sources_updated_at ON sources;
DROP TRIGGER IF EXISTS update_audit_providers_updated_at ON audit_providers;

DROP FUNCTION IF EXISTS set_audit_report_provider();
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS processes;
DROP TABLE IF EXISTS downloads;
DROP TABLE IF EXISTS audit_report_details;
DROP TABLE IF EXISTS audit_reports;
DROP TABLE IF EXISTS sources;
DROP TABLE IF EXISTS audit_providers;