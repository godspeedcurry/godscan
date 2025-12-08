package utils

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

type SpiderRecord struct {
	Url      string
	IconHash string
	ApiCount int
	UrlCount int
	CDNCount int
	CDNHosts string
	SaveDir  string
	Status   int
}

func InitSpiderDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := createSpiderTable(db); err != nil {
		return nil, err
	}
	return db, nil
}

func createSpiderTable(db *sql.DB) error {
	ddl := `
CREATE TABLE IF NOT EXISTS spider_summary (
	url TEXT PRIMARY KEY,
	icon_hash TEXT,
	api_count INTEGER,
	url_count INTEGER,
	cdn_count INTEGER,
	cdn_hosts TEXT,
	save_dir TEXT,
	status INTEGER,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_paths (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	root_url TEXT,
	source_url TEXT,
	path TEXT,
	save_dir TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sensitive_hits (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source_url TEXT,
	category TEXT,
	content TEXT,
	save_dir TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS entropy_hits (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	source_url TEXT,
	category TEXT,
	content TEXT,
	entropy REAL,
	save_dir TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS services (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	root_url TEXT,
	source_url TEXT,
	url TEXT,
	category TEXT,
	title TEXT,
	finger TEXT,
	content_type TEXT,
	status INTEGER,
	location TEXT,
	length INTEGER,
	keyword TEXT,
	simhash TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS source_maps (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	root_url TEXT,
	js_url TEXT,
	map_url TEXT,
	status INTEGER,
	length INTEGER,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cdn_hosts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	root_url TEXT,
	host TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_paths_root ON api_paths(root_url);
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_paths_root_path ON api_paths(root_url, path);
CREATE INDEX IF NOT EXISTS idx_sensitive_source ON sensitive_hits(source_url);
CREATE INDEX IF NOT EXISTS idx_entropy_source ON entropy_hits(source_url);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cdn_host ON cdn_hosts(root_url, host);
CREATE INDEX IF NOT EXISTS idx_services_root ON services(root_url);
CREATE INDEX IF NOT EXISTS idx_services_category ON services(category);
CREATE UNIQUE INDEX IF NOT EXISTS idx_source_maps_unique ON source_maps(root_url, map_url);

CREATE TABLE IF NOT EXISTS page_snapshots (
	root_url TEXT PRIMARY KEY,
	url TEXT,
	status INTEGER,
	content_type TEXT,
	headers TEXT,
	body TEXT,
	length INTEGER,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	_, err := db.Exec(ddl)
	if err != nil {
		return err
	}
	// best-effort migrations for older databases
	_, _ = db.Exec(`ALTER TABLE spider_summary ADD COLUMN cdn_count INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE spider_summary ADD COLUMN cdn_hosts TEXT DEFAULT ''`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS cdn_hosts (id INTEGER PRIMARY KEY AUTOINCREMENT, root_url TEXT, host TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_api_paths_root ON api_paths(root_url)`)
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_api_paths_root_path ON api_paths(root_url, path)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sensitive_source ON sensitive_hits(source_url)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_entropy_source ON entropy_hits(source_url)`)
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_cdn_host ON cdn_hosts(root_url, host)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS services (id INTEGER PRIMARY KEY AUTOINCREMENT, root_url TEXT, source_url TEXT, url TEXT, category TEXT, title TEXT, finger TEXT, content_type TEXT, status INTEGER, location TEXT, length INTEGER, keyword TEXT, simhash TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_services_root ON services(root_url)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_services_category ON services(category)`)
	_, _ = db.Exec(`ALTER TABLE sensitive_hits ADD COLUMN entropy REAL DEFAULT 0`)
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_sensitive_unique ON sensitive_hits(source_url, category, content)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS source_maps (id INTEGER PRIMARY KEY AUTOINCREMENT, root_url TEXT, js_url TEXT, map_url TEXT, status INTEGER, length INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_source_maps_unique ON source_maps(root_url, map_url)`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS page_snapshots (root_url TEXT PRIMARY KEY, url TEXT, status INTEGER, content_type TEXT, headers TEXT, body TEXT, length INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	return nil
}

func SaveSpiderSummary(db *sql.DB, rec SpiderRecord) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.Exec(`INSERT INTO spider_summary (url, icon_hash, api_count, url_count, cdn_count, cdn_hosts, save_dir, status, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(url) DO UPDATE SET
  icon_hash=excluded.icon_hash,
  api_count=excluded.api_count,
  url_count=excluded.url_count,
  cdn_count=excluded.cdn_count,
  cdn_hosts=excluded.cdn_hosts,
  save_dir=excluded.save_dir,
  status=excluded.status,
  updated_at=excluded.updated_at
`, rec.Url, rec.IconHash, rec.ApiCount, rec.UrlCount, rec.CDNCount, rec.CDNHosts, rec.SaveDir, rec.Status, time.Now().UTC())
	return err
}

func SaveAPIPaths(db *sql.DB, rootURL, sourceURL string, paths []string, saveDir string) error {
	if db == nil || len(paths) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO api_paths (root_url, source_url, path, save_dir) VALUES (?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, p := range paths {
		if _, err := stmt.Exec(rootURL, sourceURL, p, saveDir); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func SaveCDNHosts(db *sql.DB, rootURL string, hosts []string) error {
	if db == nil || len(hosts) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO cdn_hosts (root_url, host) VALUES (?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, h := range hosts {
		if _, err := stmt.Exec(rootURL, h); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func LoadSpiderSummaries(db *sql.DB) ([]SpiderRecord, error) {
	rows, err := db.Query(`SELECT url, icon_hash, api_count, url_count, cdn_count, cdn_hosts, save_dir, status FROM spider_summary ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SpiderRecord
	for rows.Next() {
		var r SpiderRecord
		if err := rows.Scan(&r.Url, &r.IconHash, &r.ApiCount, &r.UrlCount, &r.CDNCount, &r.CDNHosts, &r.SaveDir, &r.Status); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type APICount struct {
	Root string
	Cnt  int
}

type CDNHostRow struct {
	Root string
	Host string
}

func LoadAPICounts(db *sql.DB) ([]APICount, error) {
	rows, err := db.Query(`SELECT root_url, COUNT(*) as cnt FROM api_paths GROUP BY root_url ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APICount
	for rows.Next() {
		var r APICount
		if err := rows.Scan(&r.Root, &r.Cnt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func LoadCDNHosts(db *sql.DB) ([]CDNHostRow, error) {
	rows, err := db.Query(`SELECT root_url, host FROM cdn_hosts ORDER BY root_url, host`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CDNHostRow
	for rows.Next() {
		var r CDNHostRow
		if err := rows.Scan(&r.Root, &r.Host); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type SensitiveCount struct {
	Category string
	Count    int
}

func LoadSensitiveCounts(db *sql.DB) ([]SensitiveCount, error) {
	rows, err := db.Query(`SELECT category, COUNT(*) as cnt FROM sensitive_hits GROUP BY category ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SensitiveCount
	for rows.Next() {
		var r SensitiveCount
		if err := rows.Scan(&r.Category, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type EntropyHit struct {
	SourceURL string
	Category  string
	Content   string
	Entropy   float64
	SaveDir   string
}

type SensitiveHit struct {
	SourceURL string
	Category  string
	Content   string
	Entropy   float64
	SaveDir   string
}

func SaveSensitiveHits(db *sql.DB, hits []SensitiveHit) error {
	if db == nil || len(hits) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO sensitive_hits (source_url, category, content, entropy, save_dir) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, h := range hits {
		if _, err := stmt.Exec(h.SourceURL, h.Category, h.Content, h.Entropy, h.SaveDir); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func LoadEntropyHits(db *sql.DB) ([]EntropyHit, error) {
	rows, err := db.Query(`SELECT source_url, category, content, entropy, save_dir FROM sensitive_hits WHERE entropy > 0 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EntropyHit
	for rows.Next() {
		var e EntropyHit
		if err := rows.Scan(&e.SourceURL, &e.Category, &e.Content, &e.Entropy, &e.SaveDir); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func LoadSensitiveHits(db *sql.DB) ([]SensitiveHit, error) {
	rows, err := db.Query(`SELECT source_url, category, content, entropy, save_dir FROM sensitive_hits ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SensitiveHit
	for rows.Next() {
		var s SensitiveHit
		if err := rows.Scan(&s.SourceURL, &s.Category, &s.Content, &s.Entropy, &s.SaveDir); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type SourceMapHit struct {
	RootURL string
	JSURL   string
	MapURL  string
	Status  int
	Length  int
}

type PageSnapshot struct {
	RootURL     string
	URL         string
	Status      int
	ContentType string
	Headers     string
	Body        string
	Length      int
}

func SaveSourceMaps(db *sql.DB, hits []SourceMapHit) error {
	if db == nil || len(hits) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO source_maps (root_url, js_url, map_url, status, length) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, h := range hits {
		if _, err := stmt.Exec(h.RootURL, h.JSURL, h.MapURL, h.Status, h.Length); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func SavePageSnapshot(db *sql.DB, snap PageSnapshot) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO page_snapshots (root_url, url, status, content_type, headers, body, length, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		snap.RootURL, snap.URL, snap.Status, snap.ContentType, snap.Headers, snap.Body, snap.Length)
	return err
}

// Global DB handle used for mirroring.
var spiderDB *sql.DB

func SetSpiderDB(db *sql.DB) {
	spiderDB = db
}

func GetSpiderDB() *sql.DB {
	return spiderDB
}

// SaveService stores scan result into unified services table.
func SaveService(category string, row []string) {
	if spiderDB == nil || len(row) < 9 {
		return
	}
	status, _ := strconv.Atoi(row[4])
	length, _ := strconv.Atoi(row[6])
	_, err := spiderDB.Exec(
		`INSERT INTO services (url, category, title, finger, content_type, status, location, length, keyword, simhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row[0], category, row[1], row[2], row[3], status, row[5], length, row[7], row[8],
	)
	if err != nil {
		return
	}
}
