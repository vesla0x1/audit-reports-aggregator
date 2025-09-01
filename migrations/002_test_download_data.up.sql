WITH inserted_report AS (
    INSERT INTO audit_reports (
        source_id,
        provider_id,
        title,
        engagement_type,
        client_company,
        audit_start_date,
        audit_end_date,
        details_page_url,
        source_download_url,
        repository_url,
        summary,
        findings_summary
    )
    SELECT 
        s.id as source_id,
        p.id as provider_id,
        'Panoptic Hypovault' as title,
        'competition' as engagement_type,
        'Panoptic' as client_company,
        '2025-06-27'::DATE as audit_start_date,
        '2025-07-07'::DATE as audit_end_date,
        'https://code4rena.com/audits/2025-06-panoptic-hypovault' as details_page_url,
        'https://code4rena.com/reports/2025-06-panoptic-hypovault' as source_download_url,
        'https://github.com/code-423n4/2025-06-panoptic' as repository_url,
        'The C4 analysis yielded an aggregated total of 2 unique vulnerabilities...' as summary,
        '{"high": 2, "medium": 0, "low": 2, "informational": 0}'::jsonb as findings_summary
    FROM sources s
    JOIN audit_providers p ON s.provider_id = p.id
    WHERE p.slug = 'code4rena' 
    AND s.engagement_type = 'competition'
    RETURNING id
),
-- Insert detailed content
inserted_details AS (
    INSERT INTO audit_report_details (
        report_id,
        full_summary
    )
    SELECT 
        id,
        'The C4 analysis yielded an aggregated total of 2 unique vulnerabilities. Of these vulnerabilities, 2 received a risk rating in the category of HIGH severity and 0 received a risk rating in the category of MEDIUM severity. Additionally, C4 analysis included 2 reports detailing issues with a risk rating of LOW severity or non-critical. All of the issues presented here are linked back to their original finding, which may include relevant context from the judge and Panoptic team.' as full_summary
    FROM inserted_report
    RETURNING report_id
),
-- Create a download record
inserted_download AS (
    INSERT INTO downloads (
        report_id,
        status
    )
    SELECT 
        id,
        'pending'
    FROM inserted_report
    RETURNING id, report_id
)
-- Verify the test data
SELECT 
    ar.id as report_id,
    ar.title,
    ar.summary as brief_summary,
    ard.full_summary,
    d.id as download_id,
    d.status as download_status
FROM audit_reports ar
JOIN audit_report_details ard ON ard.report_id = ar.id
JOIN downloads d ON d.report_id = ar.id
WHERE ar.title = 'Panoptic Hypovault';