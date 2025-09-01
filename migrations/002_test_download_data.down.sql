DELETE FROM audit_reports 
WHERE title = 'Panoptic Hypovault'
-- The CASCADE constraints will automatically delete:
-- - audit_report_details (due to ON DELETE CASCADE)
-- - downloads (due to ON DELETE CASCADE)