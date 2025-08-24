package setup

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DatasetCache manages pre-generated datasets stored in SQLite
type DatasetCache struct {
	dbPath string
}

// NewDatasetCache creates a new dataset cache
func NewDatasetCache() *DatasetCache {
	cacheDir := "cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create cache directory: %v", err))
	}

	return &DatasetCache{
		dbPath: filepath.Join(cacheDir, "datasets.db"),
	}
}

// Init initializes the SQLite database and creates tables
func (dc *DatasetCache) Init() error {
	db, err := sql.Open("sqlite3", dc.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %v", err)
	}
	defer db.Close()

	// Create tables
	queries := []string{
		`CREATE TABLE IF NOT EXISTS datasets (
			id TEXT PRIMARY KEY,
			config TEXT NOT NULL,
			courses TEXT NOT NULL,
			students TEXT NOT NULL,
			enrollments TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_datasets_id ON datasets(id)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %v", query, err)
		}
	}

	return nil
}

// GetOrGenerateDataset retrieves a dataset from cache or generates and stores it
func (dc *DatasetCache) GetOrGenerateDataset(config DatasetConfig) (*Dataset, error) {
	// Create dataset ID based on config
	datasetID := fmt.Sprintf("%d_%d_%d_%d", config.Courses, config.Students, config.Enrollments, config.Capacity)

	// Try to get from cache first
	if dataset, err := dc.getFromCache(datasetID); err == nil {
		return dataset, nil
	}

	// Generate new dataset
	dataset := GenerateDataset(config)

	// Store in cache
	if err := dc.storeInCache(datasetID, config, dataset); err != nil {
		return nil, fmt.Errorf("failed to store dataset in cache: %v", err)
	}

	return dataset, nil
}

// getFromCache retrieves a dataset from SQLite cache
func (dc *DatasetCache) getFromCache(datasetID string) (*Dataset, error) {
	db, err := sql.Open("sqlite3", dc.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %v", err)
	}
	defer db.Close()

	var configJSON, coursesJSON, studentsJSON, enrollmentsJSON string
	query := `SELECT config, courses, students, enrollments FROM datasets WHERE id = ?`

	err = db.QueryRow(query, datasetID).Scan(&configJSON, &coursesJSON, &studentsJSON, &enrollmentsJSON)
	if err != nil {
		return nil, fmt.Errorf("dataset not found in cache: %v", err)
	}

	// Deserialize config
	var config DatasetConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to deserialize config: %v", err)
	}

	// Deserialize courses
	var courses []CourseData
	if err := json.Unmarshal([]byte(coursesJSON), &courses); err != nil {
		return nil, fmt.Errorf("failed to deserialize courses: %v", err)
	}

	// Deserialize students
	var students []StudentData
	if err := json.Unmarshal([]byte(studentsJSON), &students); err != nil {
		return nil, fmt.Errorf("failed to deserialize students: %v", err)
	}

	// Deserialize enrollments
	var enrollments []EnrollmentData
	if err := json.Unmarshal([]byte(enrollmentsJSON), &enrollments); err != nil {
		return nil, fmt.Errorf("failed to deserialize enrollments: %v", err)
	}

	return &Dataset{
		Config:      config,
		Courses:     courses,
		Students:    students,
		Enrollments: enrollments,
	}, nil
}

// storeInCache stores a dataset in SQLite cache
func (dc *DatasetCache) storeInCache(datasetID string, config DatasetConfig, dataset *Dataset) error {
	db, err := sql.Open("sqlite3", dc.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %v", err)
	}
	defer db.Close()

	// Serialize data
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}

	coursesJSON, err := json.Marshal(dataset.Courses)
	if err != nil {
		return fmt.Errorf("failed to serialize courses: %v", err)
	}

	studentsJSON, err := json.Marshal(dataset.Students)
	if err != nil {
		return fmt.Errorf("failed to serialize students: %v", err)
	}

	enrollmentsJSON, err := json.Marshal(dataset.Enrollments)
	if err != nil {
		return fmt.Errorf("failed to serialize enrollments: %v", err)
	}

	// Store in database
	query := `INSERT OR REPLACE INTO datasets (id, config, courses, students, enrollments) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(query, datasetID, configJSON, coursesJSON, studentsJSON, enrollmentsJSON)
	if err != nil {
		return fmt.Errorf("failed to serialize enrollments: %v", err)
	}

	return nil
}

// ClearCache removes all cached datasets
func (dc *DatasetCache) ClearCache() error {
	db, err := sql.Open("sqlite3", dc.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM datasets")
	if err != nil {
		return fmt.Errorf("failed to clear cache: %v", err)
	}

	return nil
}

// GetCacheInfo returns information about cached datasets
func (dc *DatasetCache) GetCacheInfo() (map[string]time.Time, error) {
	db, err := sql.Open("sqlite3", dc.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, created_at FROM datasets")
	if err != nil {
		return nil, fmt.Errorf("failed to query cache info: %v", err)
	}
	defer rows.Close()

	info := make(map[string]time.Time)
	for rows.Next() {
		var id string
		var createdAt time.Time
		if err := rows.Scan(&id, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan cache info: %v", err)
		}
		info[id] = createdAt
	}

	return info, nil
}



// Global cache instance
var globalCache *DatasetCache

// InitGlobalCache initializes the global dataset cache
func InitGlobalCache() error {
	globalCache = NewDatasetCache()
	return globalCache.Init()
}

// GetGlobalCache returns the global cache instance
func GetGlobalCache() *DatasetCache {
	return globalCache
}

// GetCachedDataset retrieves a dataset from the global cache
func GetCachedDataset(config DatasetConfig) (*Dataset, error) {
	if globalCache == nil {
		if err := InitGlobalCache(); err != nil {
			return nil, err
		}
	}
	return globalCache.GetOrGenerateDataset(config)
}
