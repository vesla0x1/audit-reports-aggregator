-- Create update trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 1. Audit Providers
CREATE TABLE audit_providers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    website_url TEXT,
    description TEXT,
    provider_type VARCHAR(50) CHECK (provider_type IN ('marketplace', 'firm', 'individual')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_audit_providers_updated_at BEFORE UPDATE
    ON audit_providers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 2. Sources
CREATE TABLE sources (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES audit_providers(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    engagement_type VARCHAR(100) CHECK (engagement_type IN ('competition', 'private', 'solo', 'bug_bounty')),
    index_page_url TEXT NOT NULL UNIQUE,
    
    -- Crawler tracking
    scraper_type VARCHAR(50), -- which scraper implementation handles this
    last_visited_at TIMESTAMP,
    last_reports_count INT DEFAULT 0,
    last_main_div_hash VARCHAR(64), -- detect structural changes
    
    -- Configuration
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(provider_id, name)
);

CREATE TRIGGER update_sources_updated_at BEFORE UPDATE
    ON sources FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 3. Audit Reports
CREATE TABLE audit_reports (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
    provider_id BIGINT NOT NULL REFERENCES audit_providers(id) ON DELETE RESTRICT,
    
    -- Report metadata
    title VARCHAR(500) NOT NULL,
    engagement_type VARCHAR(20) NOT NULL CHECK (engagement_type IN ('competition', 'private', 'bug_bounty', 'solo')),
    client_company VARCHAR(255),
    
    -- Audit period as proper dates
    audit_start_date DATE,
    audit_end_date DATE,
    
    -- URLs
    details_page_url TEXT NOT NULL,
    source_download_url TEXT NOT NULL,
    repository_url TEXT,
    
    -- Content
    summary TEXT,
    findings_summary JSONB,
    
    -- Tracking
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE(source_id, details_page_url)
);

CREATE TRIGGER update_audit_reports_updated_at BEFORE UPDATE
    ON audit_reports FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Auto-set provider_id from source
CREATE OR REPLACE FUNCTION set_audit_report_provider()
RETURNS TRIGGER AS $$
BEGIN
    SELECT provider_id INTO NEW.provider_id
    FROM sources WHERE id = NEW.source_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_report_provider_trigger
    BEFORE INSERT OR UPDATE OF source_id ON audit_reports
    FOR EACH ROW EXECUTE FUNCTION set_audit_report_provider();

-- 4. Audit Report Details (for lengthy content)
CREATE TABLE audit_report_details (
    id BIGSERIAL PRIMARY KEY,
    report_id BIGINT NOT NULL UNIQUE REFERENCES audit_reports(id) ON DELETE CASCADE,
    
    -- Full content
    full_summary TEXT,     -- Complete audit summary/description
    raw_content JSONB,     -- Any other scraped content
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_audit_report_details_updated_at BEFORE UPDATE
    ON audit_report_details FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 5. Downloads
CREATE TABLE downloads (
    id BIGSERIAL PRIMARY KEY,
    report_id BIGINT NOT NULL UNIQUE REFERENCES audit_reports(id) ON DELETE CASCADE,
    
    -- Storage information (minimal)
    storage_path VARCHAR(500), -- e.g., "code4rena/1234_some_audit.pdf"
    file_hash VARCHAR(64), -- SHA-256 for integrity and deduplication
    file_extension VARCHAR(10),
    
    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'in_progress', 'completed', 'failed')),
    error_message TEXT,
    attempt_count INT DEFAULT 0 CHECK (attempt_count >= 0),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_downloads_updated_at BEFORE UPDATE
    ON downloads FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 6. Processes
CREATE TABLE processes (
    id BIGSERIAL PRIMARY KEY,
    download_id BIGINT NOT NULL UNIQUE REFERENCES downloads(id) ON DELETE CASCADE,
    
    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'in_progress', 'completed', 'failed')),
    error_message TEXT,
    attempt_count INT DEFAULT 0 CHECK (attempt_count >= 0),
    
    -- Processing metadata (minimal)
    processor_version VARCHAR(50), -- might be useful for reprocessing
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_processes_updated_at BEFORE UPDATE
    ON processes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 7. Create Indexes

-- Provider indexes
CREATE INDEX idx_audit_providers_active ON audit_providers(is_active) WHERE is_active = true;
CREATE INDEX idx_audit_providers_slug ON audit_providers(slug);

-- Source indexes
CREATE INDEX idx_sources_active ON sources(is_active) WHERE is_active = true;
CREATE INDEX idx_sources_provider_active ON sources(provider_id, is_active);
CREATE INDEX idx_sources_last_visited ON sources(last_visited_at);

-- Audit report indexes
CREATE INDEX idx_audit_reports_source_id ON audit_reports(source_id);
CREATE INDEX idx_audit_reports_provider_id ON audit_reports(provider_id);
CREATE INDEX idx_audit_reports_engagement_type ON audit_reports(engagement_type);
CREATE INDEX idx_audit_reports_client_company ON audit_reports(client_company);
CREATE INDEX idx_audit_reports_created_at ON audit_reports(created_at DESC);

CREATE INDEX idx_audit_reports_start_date ON audit_reports(audit_start_date);
CREATE INDEX idx_audit_reports_end_date ON audit_reports(audit_end_date);
CREATE INDEX idx_audit_reports_date_range ON audit_reports(audit_start_date, audit_end_date);

--- Audit report details
CREATE INDEX idx_audit_report_details_report_id ON audit_report_details(report_id);

-- Download indexes
CREATE INDEX idx_downloads_report_id ON downloads(report_id);
CREATE INDEX idx_downloads_status ON downloads(status);
CREATE INDEX idx_downloads_pending ON downloads(status) WHERE status = 'pending';
CREATE INDEX idx_downloads_failed_retry ON downloads(status, attempt_count) WHERE status = 'failed';

-- Process indexes
CREATE INDEX idx_processes_download_id ON processes(download_id);
CREATE INDEX idx_processes_status ON processes(status);
CREATE INDEX idx_processes_pending ON processes(status) WHERE status = 'pending';

-- 7. Insert seed data
-- Insert audit providers
INSERT INTO audit_providers (name, slug, website_url, provider_type, description, is_active) VALUES 
    ('Cantina', 'cantina', 'https://cantina.xyz', 'marketplace', 
     'Competitive audit marketplace combining crowdsourced security researchers with managed audit competitions', 
     false),
    
    ('Code4rena', 'code4rena', 'https://code4rena.com', 'marketplace', 
     'Leading competitive audit platform connecting projects with security researchers through audit contests', 
     true),
    
    ('Sherlock', 'sherlock', 'https://sherlock.xyz', 'marketplace', 
     'Audit marketplace with a unique coverage protocol that offers exploit protection alongside audit competitions', 
     false),
    
    ('CodeHawks', 'codehawks', 'https://codehawks.com', 'marketplace', 
     'Competitive auditing platform by Cyfrin focusing on smart contract security through community-driven contests', 
     false),

     ('Zellic', 'zellic', 'https://zellic.io', 'firm', 
     'Specialized security firm focused on in-depth audits for DeFi, zero-knowledge systems, and blockchain infrastructure', 
     false),
    
    ('OpenZeppelin', 'openzeppelin', 'https://openzeppelin.com', 'firm', 
     'Industry-leading security company known for secure smart contract libraries and comprehensive security audits', 
     false)
ON CONFLICT (name) DO NOTHING;

-- Insert sources
-- Code4rena (active)
INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'Code4rena Contests', 'competition', 'https://code4rena.com/reports', true
FROM audit_providers p WHERE p.slug = 'code4rena'
ON CONFLICT (index_page_url) DO NOTHING;

-- Cantina sources (inactive)
INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'Cantina Reviews', 'private', 'https://cantina.xyz/portfolio', false
FROM audit_providers p WHERE p.slug = 'cantina'
ON CONFLICT (index_page_url) DO NOTHING;

INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'Cantina Competitions', 'competition', 'https://cantina.xyz/portfolio?section=cantina-competitions', false
FROM audit_providers p WHERE p.slug = 'cantina'
ON CONFLICT (index_page_url) DO NOTHING;

INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'Cantina Solo', 'solo', 'https://cantina.xyz/portfolio?section=cantina-solo', false
FROM audit_providers p WHERE p.slug = 'cantina'
ON CONFLICT (index_page_url) DO NOTHING;

INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'Spearbit Guild', 'private', 'https://cantina.xyz/portfolio?section=spearbit-guild', false
FROM audit_providers p WHERE p.slug = 'cantina'
ON CONFLICT (index_page_url) DO NOTHING;

-- Sherlock (inactive)
INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'Sherlock Contests', 'competition', 'https://audits.sherlock.xyz/api/contests?per_page=1000', false
FROM audit_providers p WHERE p.slug = 'sherlock'
ON CONFLICT (index_page_url) DO NOTHING;

-- CodeHawks (inactive)
INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'CodeHawks Contests', 'competition', 'https://codehawks.cyfrin.io/contests', false
FROM audit_providers p WHERE p.slug = 'codehawks'
ON CONFLICT (index_page_url) DO NOTHING;

-- Zellic (inactive for now)
INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'Zellic Reports', 'private', 'https://github.com/Zellic/publications', false
FROM audit_providers p WHERE p.slug = 'zellic'
ON CONFLICT (index_page_url) DO NOTHING;

-- OpenZeppelin (inactive for now)
INSERT INTO sources (provider_id, name, engagement_type, index_page_url, is_active)
SELECT p.id, 'OpenZeppelin Reports', 'private', 'https://blog.openzeppelin.com/security-audits', false
FROM audit_providers p WHERE p.slug = 'openzeppelin'
ON CONFLICT (index_page_url) DO NOTHING;

-- Verify what was inserted
SELECT 
    ap.name as provider,
    ap.provider_type,
    ap.is_active as provider_active,
    COUNT(s.id) as source_count
FROM audit_providers ap
LEFT JOIN sources s ON s.provider_id = ap.id
GROUP BY ap.id, ap.name, ap.provider_type, ap.is_active
ORDER BY ap.is_active DESC, ap.name;