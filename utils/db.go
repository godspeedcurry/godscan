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

CREATE TABLE IF NOT EXISTS finger_results (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT,
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

CREATE TABLE IF NOT EXISTS dirbrute_results (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT,
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

CREATE TABLE IF NOT EXISTS cdn_hosts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	root_url TEXT,
	host TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`
	_, err := db.Exec(ddl)
	if err != nil {
		return err
	}
	// best-effort migrations for older databases
	_, _ = db.Exec(`ALTER TABLE spider_summary ADD COLUMN cdn_count INTEGER DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE spider_summary ADD COLUMN cdn_hosts TEXT DEFAULT ''`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS cdn_hosts (id INTEGER PRIMARY KEY AUTOINCREMENT, root_url TEXT, host TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
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

func SaveSensitiveHits(db *sql.DB, sourceURL, category string, contents []string, saveDir string) error {
	if db == nil || len(contents) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO sensitive_hits (source_url, category, content, save_dir) VALUES (?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, c := range contents {
		if _, err := stmt.Exec(sourceURL, category, c, saveDir); err != nil {
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

func SaveEntropyHits(db *sql.DB, sourceURL, category, saveDir string, data []EntropyHit) error {
	if db == nil || len(data) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO entropy_hits (source_url, category, content, entropy, save_dir) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, d := range data {
		if _, err := stmt.Exec(sourceURL, category, d.Content, d.Entropy, saveDir); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func LoadEntropyHits(db *sql.DB) ([]EntropyHit, error) {
	rows, err := db.Query(`SELECT source_url, category, content, entropy, save_dir FROM entropy_hits ORDER BY created_at DESC`)
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

// Global DB handle used for CSV mirroring.
var spiderDB *sql.DB

func SetSpiderDB(db *sql.DB) {
	spiderDB = db
}

func SaveFingerLike(table string, row []string) {
	if spiderDB == nil || len(row) < 9 {
		return
	}
	status, _ := strconv.Atoi(row[4])
	length, _ := strconv.Atoi(row[6])
	_, err := spiderDB.Exec(
		fmt.Sprintf(`INSERT INTO %s (url, title, finger, content_type, status, location, length, keyword, simhash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, table),
		row[0], row[1], row[2], row[3], status, row[5], length, row[7], row[8],
	)
	if err != nil {
		// silent to avoid noisy logs in hot path
		return
	}
}
