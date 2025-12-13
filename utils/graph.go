package utils

import (
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
	"sync"
)

type GraphEdge struct {
	Root  string `json:"root"`
	From  string `json:"from"`
	To    string `json:"to"`
	Depth int    `json:"depth"`
}

type GraphCollector struct {
	mu    sync.Mutex
	edges []GraphEdge
	limit int
}

var defaultGraphCollector = &GraphCollector{limit: 5000}

func GetGraphCollector() *GraphCollector {
	return defaultGraphCollector
}

func (g *GraphCollector) SetLimit(n int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if n > 0 {
		g.limit = n
	}
}

func (g *GraphCollector) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.edges = nil
}

func (g *GraphCollector) AddEdge(root, from, to string, depth int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.limit > 0 && len(g.edges) >= g.limit {
		return
	}
	g.edges = append(g.edges, GraphEdge{Root: root, From: from, To: to, Depth: depth})
}

func (g *GraphCollector) Snapshot() []GraphEdge {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]GraphEdge, len(g.edges))
	copy(out, g.edges)
	return out
}

func SaveGraphJSON(path string, edges []GraphEdge) error {
	data, err := json.Marshal(edges)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadGraphJSON(path string) ([]GraphEdge, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var edges []GraphEdge
	if err := json.Unmarshal(b, &edges); err != nil {
		return nil, err
	}
	return edges, nil
}

func SaveGraphDB(db *sql.DB, edges []GraphEdge) error {
	if db == nil || len(edges) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO graph_edges (root_url, from_url, to_url, depth) VALUES (?, ?, ?, ?) ON CONFLICT(root_url, from_url, to_url) DO NOTHING`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, e := range edges {
		if _, err := stmt.Exec(e.Root, e.From, e.To, e.Depth); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func LoadGraphDB(db *sql.DB, limit int) ([]GraphEdge, error) {
	if db == nil {
		return nil, nil
	}
	q := `SELECT root_url, from_url, to_url, depth FROM graph_edges ORDER BY created_at DESC`
	if limit > 0 {
		q += " LIMIT " + strconv.Itoa(limit)
	}
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GraphEdge
	for rows.Next() {
		var e GraphEdge
		if err := rows.Scan(&e.Root, &e.From, &e.To, &e.Depth); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
