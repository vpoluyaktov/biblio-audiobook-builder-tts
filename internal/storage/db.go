package storage

import (
	"abb_tts/internal/logger"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the SQLite database connection
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// Config represents application configuration stored in the database
type Config struct {
	// Basic settings
	LogFile         string `json:"log_file"`
	OutputDir       string `json:"output_dir"`
	TempDir         string `json:"temp_dir"`
	DefaultVoice    string `json:"default_voice"`
	DefaultProvider string `json:"default_provider"`

	// Server settings
	ServerPort  string `json:"server_port"`
	ServerHost  string `json:"server_host"`
	OpenBrowser bool   `json:"open_browser"`

	// TTS settings
	BitRateKbs              int     `json:"bit_rate_kbs"`
	SampleRateHz            int     `json:"sample_rate_hz"`
	DefaultSpeed            float64 `json:"default_speed"`
	DefaultPitch            float64 `json:"default_pitch"`
	ChapterGapSeconds       int     `json:"chapter_gap_seconds"`
	PronunciationDictFile   string  `json:"pronunciation_dict_file"`
	UseDefaultPronunciation bool    `json:"use_default_pronunciation"`
	MaxFileSizeMB           int     `json:"max_file_size_mb"`

	// Performance settings
	ConcurrentTTSWorkers int            `json:"concurrent_tts_workers"`
	ProviderTTSWorkers   map[string]int `json:"provider_tts_workers"`
	ConcurrentEncoders   int            `json:"concurrent_encoders"`

	// Cloud provider settings
	CloudAPIKey       string `json:"cloud_api_key"`
	GoogleTTSEndpoint string `json:"google_tts_endpoint"`
	AzureTTSEndpoint  string `json:"azure_tts_endpoint"`
	GoogleAPIKey      string `json:"google_api_key"`

	// OpenTTS settings
	OpenTTSURL string `json:"opentts_url"`

	// RHVoice settings
	RHVoiceURL string `json:"rhvoice_url"`

	// Silero TTS settings
	SileroURL string `json:"silero_url"`

	// OpenAI TTS settings
	OpenAIAPIKey string `json:"openai_api_key"`

	// Azure TTS settings
	AzureTTSKey    string `json:"azure_tts_key"`
	AzureTTSRegion string `json:"azure_tts_region"`

	// Audiobookshelf integration
	AudiobookshelfURL      string `json:"audiobookshelf_url"`
	AudiobookshelfUser     string `json:"audiobookshelf_user"`
	AudiobookshelfPassword string `json:"audiobookshelf_password"`
	AudiobookshelfLibrary  string `json:"audiobookshelf_library"`
}

// WorkerProgress represents the progress of a single parallel worker
type WorkerProgress struct {
	WorkerID       int     `json:"worker_id"`
	ChapterIndex   int     `json:"chapter_index"`
	ChapterTitle   string  `json:"chapter_title"`
	Progress       float64 `json:"progress"`
	ChunksTotal    int     `json:"chunks_total"`
	ChunksComplete int     `json:"chunks_complete"`
	Active         bool    `json:"active"`
}

// Job represents a conversion job stored in the database
type Job struct {
	ID                 string           `json:"id"`
	Status             string           `json:"status"`
	FileName           string           `json:"file_name"`
	FilePath           string           `json:"file_path"`
	Provider           string           `json:"provider"`
	Voice              string           `json:"voice"`
	Speed              float64          `json:"speed"`
	Pitch              float64          `json:"pitch"`
	BookTitle          string           `json:"book_title"`
	BookAuthor         string           `json:"book_author"`
	OutputPath         string           `json:"output_path"`
	M4BFile            string           `json:"m4b_file"`
	M4BFiles           []string         `json:"m4b_files"`
	ConversionProgress float64          `json:"conversion_progress"`
	BuildProgress      float64          `json:"build_progress"`
	CurrentChapter     string           `json:"current_chapter"`
	TotalChapters      int              `json:"total_chapters"`
	CurrentChapterNum  int              `json:"current_chapter_num"`
	WorkerProgress     []WorkerProgress `json:"worker_progress,omitempty"`
	NumWorkers         int              `json:"num_workers,omitempty"`
	Error              string           `json:"error"`
	CreatedAt          time.Time        `json:"created_at"`
	StartedAt          *time.Time       `json:"started_at"`
	CompletedAt        *time.Time       `json:"completed_at"`
}

// NewDB creates a new database connection
func NewDB(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate creates the database schema
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		file_name TEXT NOT NULL,
		file_path TEXT,
		provider TEXT,
		voice TEXT,
		speed REAL DEFAULT 1.0,
		pitch REAL DEFAULT 1.0,
		book_title TEXT,
		book_author TEXT,
		output_path TEXT,
		m4b_file TEXT,
		m4b_files TEXT,
		conversion_progress REAL DEFAULT 0,
		build_progress REAL DEFAULT 0,
		current_chapter TEXT,
		total_chapters INTEGER DEFAULT 0,
		current_chapter_num INTEGER DEFAULT 0,
		worker_progress TEXT,
		num_workers INTEGER DEFAULT 0,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);

	CREATE TABLE IF NOT EXISTS opds_sources (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		description TEXT,
		username TEXT DEFAULT '',
		password TEXT DEFAULT '',
		is_default BOOLEAN DEFAULT 0,
		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_opds_sources_enabled ON opds_sources(enabled);

	CREATE TABLE IF NOT EXISTS nouns (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lang TEXT NOT NULL,
		noun TEXT NOT NULL,
		gender TEXT NOT NULL,
		form TEXT NOT NULL,
		singular TEXT NOT NULL,
		plurals TEXT NOT NULL,
		is_custom BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(lang, noun)
	);

	CREATE INDEX IF NOT EXISTS idx_nouns_lang ON nouns(lang);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Add worker_progress and num_workers columns if they don't exist (migration for existing DBs)
	db.conn.Exec("ALTER TABLE jobs ADD COLUMN worker_progress TEXT")
	db.conn.Exec("ALTER TABLE jobs ADD COLUMN num_workers INTEGER DEFAULT 0")

	return nil
}

// GetConfig retrieves a configuration value
func (db *DB) GetConfig(key string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var value string
	err := db.conn.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetConfig sets a configuration value
func (db *DB) SetConfig(key, value string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		INSERT INTO config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

// GetAllConfig retrieves all configuration as a Config struct
func (db *DB) GetAllConfig() (*Config, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.conn.Query("SELECT key, value FROM config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configMap := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		configMap[key] = value
	}

	// Convert map to Config struct with defaults
	cfg := DefaultConfig()

	if v, ok := configMap["log_file"]; ok {
		cfg.LogFile = v
	}
	if v, ok := configMap["output_dir"]; ok {
		cfg.OutputDir = v
	}
	if v, ok := configMap["temp_dir"]; ok {
		cfg.TempDir = v
	}
	if v, ok := configMap["default_voice"]; ok {
		cfg.DefaultVoice = v
	}
	if v, ok := configMap["default_provider"]; ok {
		cfg.DefaultProvider = v
	}
	if v, ok := configMap["server_port"]; ok {
		cfg.ServerPort = v
	}
	if v, ok := configMap["server_host"]; ok {
		cfg.ServerHost = v
	}
	if v, ok := configMap["open_browser"]; ok {
		cfg.OpenBrowser = v == "true"
	}
	if v, ok := configMap["bit_rate_kbs"]; ok {
		fmt.Sscanf(v, "%d", &cfg.BitRateKbs)
	}
	if v, ok := configMap["sample_rate_hz"]; ok {
		fmt.Sscanf(v, "%d", &cfg.SampleRateHz)
	}
	if v, ok := configMap["default_speed"]; ok {
		fmt.Sscanf(v, "%f", &cfg.DefaultSpeed)
	}
	if v, ok := configMap["default_pitch"]; ok {
		fmt.Sscanf(v, "%f", &cfg.DefaultPitch)
	}
	if v, ok := configMap["chapter_gap_seconds"]; ok {
		fmt.Sscanf(v, "%d", &cfg.ChapterGapSeconds)
	}
	if v, ok := configMap["pronunciation_dict_file"]; ok {
		cfg.PronunciationDictFile = v
	}
	if v, ok := configMap["use_default_pronunciation"]; ok {
		cfg.UseDefaultPronunciation = v == "true"
	}
	if v, ok := configMap["max_file_size_mb"]; ok {
		fmt.Sscanf(v, "%d", &cfg.MaxFileSizeMB)
	}
	if v, ok := configMap["concurrent_tts_workers"]; ok {
		fmt.Sscanf(v, "%d", &cfg.ConcurrentTTSWorkers)
	}
	// Load per-provider TTS workers
	cfg.ProviderTTSWorkers = make(map[string]int)
	providers := []string{"espeak", "google", "opentts", "rhvoice", "silero", "openai", "azure"}
	for _, p := range providers {
		if v, ok := configMap["tts_workers_"+p]; ok {
			var workers int
			fmt.Sscanf(v, "%d", &workers)
			cfg.ProviderTTSWorkers[p] = workers
		}
	}
	if v, ok := configMap["concurrent_encoders"]; ok {
		fmt.Sscanf(v, "%d", &cfg.ConcurrentEncoders)
	}
	if v, ok := configMap["cloud_api_key"]; ok {
		cfg.CloudAPIKey = v
	}
	if v, ok := configMap["google_tts_endpoint"]; ok {
		cfg.GoogleTTSEndpoint = v
	}
	if v, ok := configMap["azure_tts_endpoint"]; ok {
		cfg.AzureTTSEndpoint = v
	}
	if v, ok := configMap["google_api_key"]; ok {
		cfg.GoogleAPIKey = v
	}
	if v, ok := configMap["opentts_url"]; ok {
		cfg.OpenTTSURL = v
	}
	if v, ok := configMap["rhvoice_url"]; ok {
		cfg.RHVoiceURL = v
	}
	if v, ok := configMap["silero_url"]; ok {
		cfg.SileroURL = v
	}
	if v, ok := configMap["openai_api_key"]; ok {
		cfg.OpenAIAPIKey = v
	}
	if v, ok := configMap["azure_tts_key"]; ok {
		cfg.AzureTTSKey = v
	}
	if v, ok := configMap["azure_tts_region"]; ok {
		cfg.AzureTTSRegion = v
	}
	if v, ok := configMap["audiobookshelf_url"]; ok {
		cfg.AudiobookshelfURL = v
	}
	if v, ok := configMap["audiobookshelf_user"]; ok {
		cfg.AudiobookshelfUser = v
	}
	if v, ok := configMap["audiobookshelf_password"]; ok {
		cfg.AudiobookshelfPassword = v
	}
	if v, ok := configMap["audiobookshelf_library"]; ok {
		cfg.AudiobookshelfLibrary = v
	}

	return cfg, nil
}

// SaveAllConfig saves all configuration from a Config struct
func (db *DB) SaveAllConfig(cfg *Config) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	configs := map[string]string{
		"log_file":                  cfg.LogFile,
		"output_dir":                cfg.OutputDir,
		"temp_dir":                  cfg.TempDir,
		"default_voice":             cfg.DefaultVoice,
		"default_provider":          cfg.DefaultProvider,
		"server_port":               cfg.ServerPort,
		"server_host":               cfg.ServerHost,
		"open_browser":              fmt.Sprintf("%t", cfg.OpenBrowser),
		"bit_rate_kbs":              fmt.Sprintf("%d", cfg.BitRateKbs),
		"sample_rate_hz":            fmt.Sprintf("%d", cfg.SampleRateHz),
		"default_speed":             fmt.Sprintf("%.2f", cfg.DefaultSpeed),
		"default_pitch":             fmt.Sprintf("%.2f", cfg.DefaultPitch),
		"chapter_gap_seconds":       fmt.Sprintf("%d", cfg.ChapterGapSeconds),
		"pronunciation_dict_file":   cfg.PronunciationDictFile,
		"use_default_pronunciation": fmt.Sprintf("%t", cfg.UseDefaultPronunciation),
		"max_file_size_mb":          fmt.Sprintf("%d", cfg.MaxFileSizeMB),
		"cloud_api_key":             cfg.CloudAPIKey,
		"google_tts_endpoint":       cfg.GoogleTTSEndpoint,
		"azure_tts_endpoint":        cfg.AzureTTSEndpoint,
		"google_api_key":            cfg.GoogleAPIKey,
		"opentts_url":               cfg.OpenTTSURL,
		"openai_api_key":            cfg.OpenAIAPIKey,
		"azure_tts_key":             cfg.AzureTTSKey,
		"azure_tts_region":          cfg.AzureTTSRegion,
		"audiobookshelf_url":        cfg.AudiobookshelfURL,
		"audiobookshelf_user":       cfg.AudiobookshelfUser,
		"audiobookshelf_password":   cfg.AudiobookshelfPassword,
		"audiobookshelf_library":    cfg.AudiobookshelfLibrary,
	}

	for key, value := range configs {
		if _, err := stmt.Exec(key, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		LogFile:                 "abb_tts.log",
		OutputDir:               "./output",
		TempDir:                 "./temp",
		DefaultVoice:            "en-US",
		DefaultProvider:         "espeak",
		ServerPort:              "8080",
		ServerHost:              "0.0.0.0",
		OpenBrowser:             true,
		BitRateKbs:              128,
		SampleRateHz:            44100,
		DefaultSpeed:            1.0,
		DefaultPitch:            1.0,
		ChapterGapSeconds:       2,
		PronunciationDictFile:   "",
		UseDefaultPronunciation: true,
		MaxFileSizeMB:           2000,
		CloudAPIKey:             "",
		GoogleTTSEndpoint:       "",
		AzureTTSEndpoint:        "",
		GoogleAPIKey:            "",
		OpenTTSURL:              "",
		OpenAIAPIKey:            "",
		AzureTTSKey:             "",
		AzureTTSRegion:          "",
		AudiobookshelfURL:       "",
		AudiobookshelfUser:      "admin",
		AudiobookshelfPassword:  "",
		AudiobookshelfLibrary:   "TTS books",
	}
}

// Job methods

// CreateJob creates a new job in the database
func (db *DB) CreateJob(job *Job) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	m4bFilesJSON, _ := json.Marshal(job.M4BFiles)
	workerProgressJSON, _ := json.Marshal(job.WorkerProgress)

	_, err := db.conn.Exec(`
		INSERT INTO jobs (id, status, file_name, file_path, provider, voice, speed, pitch,
			book_title, book_author, output_path, m4b_file, m4b_files, conversion_progress, build_progress,
			current_chapter, total_chapters, current_chapter_num, worker_progress, num_workers, error, created_at, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.Status, job.FileName, job.FilePath, job.Provider, job.Voice,
		job.Speed, job.Pitch, job.BookTitle, job.BookAuthor, job.OutputPath,
		job.M4BFile, string(m4bFilesJSON), job.ConversionProgress, job.BuildProgress, job.CurrentChapter,
		job.TotalChapters, job.CurrentChapterNum, string(workerProgressJSON), job.NumWorkers, job.Error, job.CreatedAt,
		job.StartedAt, job.CompletedAt)

	return err
}

// UpdateJob updates an existing job
func (db *DB) UpdateJob(job *Job) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	m4bFilesJSON, _ := json.Marshal(job.M4BFiles)
	workerProgressJSON, _ := json.Marshal(job.WorkerProgress)

	_, err := db.conn.Exec(`
		UPDATE jobs SET status = ?, file_name = ?, file_path = ?, provider = ?, voice = ?,
			speed = ?, pitch = ?, book_title = ?, book_author = ?, output_path = ?,
			m4b_file = ?, m4b_files = ?, conversion_progress = ?, build_progress = ?, current_chapter = ?,
			total_chapters = ?, current_chapter_num = ?, worker_progress = ?, num_workers = ?, error = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`, job.Status, job.FileName, job.FilePath, job.Provider, job.Voice,
		job.Speed, job.Pitch, job.BookTitle, job.BookAuthor, job.OutputPath,
		job.M4BFile, string(m4bFilesJSON), job.ConversionProgress, job.BuildProgress, job.CurrentChapter,
		job.TotalChapters, job.CurrentChapterNum, string(workerProgressJSON), job.NumWorkers, job.Error, job.StartedAt,
		job.CompletedAt, job.ID)

	return err
}

// GetJob retrieves a job by ID
func (db *DB) GetJob(id string) (*Job, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	job := &Job{}
	var m4bFilesJSON string
	var workerProgressJSON sql.NullString
	var startedAt, completedAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, status, file_name, file_path, provider, voice, speed, pitch,
			book_title, book_author, output_path, m4b_file, m4b_files, conversion_progress, build_progress,
			current_chapter, total_chapters, current_chapter_num, worker_progress, num_workers, error, created_at, started_at, completed_at
		FROM jobs WHERE id = ?
	`, id).Scan(&job.ID, &job.Status, &job.FileName, &job.FilePath, &job.Provider,
		&job.Voice, &job.Speed, &job.Pitch, &job.BookTitle, &job.BookAuthor,
		&job.OutputPath, &job.M4BFile, &m4bFilesJSON, &job.ConversionProgress, &job.BuildProgress,
		&job.CurrentChapter, &job.TotalChapters, &job.CurrentChapterNum,
		&workerProgressJSON, &job.NumWorkers, &job.Error, &job.CreatedAt, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	json.Unmarshal([]byte(m4bFilesJSON), &job.M4BFiles)
	if workerProgressJSON.Valid {
		json.Unmarshal([]byte(workerProgressJSON.String), &job.WorkerProgress)
	}

	return job, nil
}

// ListJobs retrieves all jobs, optionally filtered by status
func (db *DB) ListJobs(status string, limit int) ([]*Job, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT id, status, file_name, file_path, provider, voice, speed, pitch,
			book_title, book_author, output_path, m4b_file, m4b_files, conversion_progress, build_progress,
			current_chapter, total_chapters, current_chapter_num, worker_progress, num_workers, error, created_at, started_at, completed_at
		FROM jobs
	`
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		var m4bFilesJSON string
		var workerProgressJSON sql.NullString
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(&job.ID, &job.Status, &job.FileName, &job.FilePath, &job.Provider,
			&job.Voice, &job.Speed, &job.Pitch, &job.BookTitle, &job.BookAuthor,
			&job.OutputPath, &job.M4BFile, &m4bFilesJSON, &job.ConversionProgress, &job.BuildProgress,
			&job.CurrentChapter, &job.TotalChapters, &job.CurrentChapterNum,
			&workerProgressJSON, &job.NumWorkers, &job.Error, &job.CreatedAt, &startedAt, &completedAt)
		if err != nil {
			return nil, err
		}

		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}

		json.Unmarshal([]byte(m4bFilesJSON), &job.M4BFiles)
		if workerProgressJSON.Valid {
			json.Unmarshal([]byte(workerProgressJSON.String), &job.WorkerProgress)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// DeleteJob deletes a job by ID
func (db *DB) DeleteJob(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM jobs WHERE id = ?", id)
	return err
}

// GetPendingJob retrieves the oldest pending job (FIFO)
func (db *DB) GetPendingJob() (*Job, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	job := &Job{}
	var m4bFilesJSON string
	var startedAt, completedAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, status, file_name, file_path, provider, voice, speed, pitch,
			book_title, book_author, output_path, m4b_file, m4b_files, conversion_progress, build_progress,
			current_chapter, total_chapters, current_chapter_num, error, created_at, started_at, completed_at
		FROM jobs WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&job.ID, &job.Status, &job.FileName, &job.FilePath, &job.Provider,
		&job.Voice, &job.Speed, &job.Pitch, &job.BookTitle, &job.BookAuthor,
		&job.OutputPath, &job.M4BFile, &m4bFilesJSON, &job.ConversionProgress, &job.BuildProgress,
		&job.CurrentChapter, &job.TotalChapters, &job.CurrentChapterNum,
		&job.Error, &job.CreatedAt, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	json.Unmarshal([]byte(m4bFilesJSON), &job.M4BFiles)

	return job, nil
}

// CleanupOldJobs removes jobs older than the specified duration
func (db *DB) CleanupOldJobs(olderThan time.Duration) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	result, err := db.conn.Exec("DELETE FROM jobs WHERE created_at < ? AND status IN ('completed', 'failed', 'cancelled')", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Noun represents a noun entry for number normalization
type Noun struct {
	ID        int64     `json:"id"`
	Lang      string    `json:"lang"`
	Noun      string    `json:"noun"`
	Gender    string    `json:"gender"` // m, f, n
	Form      string    `json:"form"`   // o (ordinal), c (cardinal)
	Singular  string    `json:"singular"`
	Plurals   string    `json:"plurals"`   // pipe-separated
	IsCustom  bool      `json:"is_custom"` // true if user-added, false if from defaults
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateNoun creates a new noun in the database
func (db *DB) CreateNoun(noun *Noun) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	result, err := db.conn.Exec(`
		INSERT INTO nouns (lang, noun, gender, form, singular, plurals, is_custom, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(lang, noun) DO UPDATE SET 
			gender = excluded.gender,
			form = excluded.form,
			singular = excluded.singular,
			plurals = excluded.plurals,
			is_custom = excluded.is_custom,
			updated_at = excluded.updated_at
	`, noun.Lang, noun.Noun, noun.Gender, noun.Form, noun.Singular, noun.Plurals,
		noun.IsCustom, noun.CreatedAt, noun.UpdatedAt)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	noun.ID = id
	return nil
}

// UpdateNoun updates an existing noun
func (db *DB) UpdateNoun(noun *Noun) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		UPDATE nouns SET gender = ?, form = ?, singular = ?, plurals = ?, is_custom = ?, updated_at = ?
		WHERE id = ?
	`, noun.Gender, noun.Form, noun.Singular, noun.Plurals, noun.IsCustom, time.Now(), noun.ID)

	return err
}

// GetNoun retrieves a noun by ID
func (db *DB) GetNoun(id int64) (*Noun, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	noun := &Noun{}
	err := db.conn.QueryRow(`
		SELECT id, lang, noun, gender, form, singular, plurals, is_custom, created_at, updated_at
		FROM nouns WHERE id = ?
	`, id).Scan(&noun.ID, &noun.Lang, &noun.Noun, &noun.Gender, &noun.Form,
		&noun.Singular, &noun.Plurals, &noun.IsCustom, &noun.CreatedAt, &noun.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return noun, nil
}

// GetNounByLangAndWord retrieves a noun by language and word
func (db *DB) GetNounByLangAndWord(lang, word string) (*Noun, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	noun := &Noun{}
	err := db.conn.QueryRow(`
		SELECT id, lang, noun, gender, form, singular, plurals, is_custom, created_at, updated_at
		FROM nouns WHERE lang = ? AND noun = ?
	`, lang, word).Scan(&noun.ID, &noun.Lang, &noun.Noun, &noun.Gender, &noun.Form,
		&noun.Singular, &noun.Plurals, &noun.IsCustom, &noun.CreatedAt, &noun.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return noun, nil
}

// ListNouns retrieves all nouns, optionally filtered by language
func (db *DB) ListNouns(lang string) ([]*Noun, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `SELECT id, lang, noun, gender, form, singular, plurals, is_custom, created_at, updated_at FROM nouns`
	args := []interface{}{}

	if lang != "" {
		query += " WHERE lang = ?"
		args = append(args, lang)
	}

	query += " ORDER BY lang, noun"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nouns []*Noun
	for rows.Next() {
		noun := &Noun{}
		err := rows.Scan(&noun.ID, &noun.Lang, &noun.Noun, &noun.Gender, &noun.Form,
			&noun.Singular, &noun.Plurals, &noun.IsCustom, &noun.CreatedAt, &noun.UpdatedAt)
		if err != nil {
			return nil, err
		}
		nouns = append(nouns, noun)
	}

	return nouns, nil
}

// DeleteNoun deletes a noun by ID
func (db *DB) DeleteNoun(id int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM nouns WHERE id = ?", id)
	return err
}

// DeleteNounByLangAndWord deletes a noun by language and word
func (db *DB) DeleteNounByLangAndWord(lang, word string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM nouns WHERE lang = ? AND noun = ?", lang, word)
	return err
}

// GetNounLanguages returns all unique languages in the nouns table
func (db *DB) GetNounLanguages() ([]string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.conn.Query("SELECT DISTINCT lang FROM nouns ORDER BY lang")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var langs []string
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err != nil {
			return nil, err
		}
		langs = append(langs, lang)
	}

	return langs, nil
}

// CountNouns returns the number of nouns for a language
func (db *DB) CountNouns(lang string) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var count int
	query := "SELECT COUNT(*) FROM nouns"
	args := []interface{}{}

	if lang != "" {
		query += " WHERE lang = ?"
		args = append(args, lang)
	}

	err := db.conn.QueryRow(query, args...).Scan(&count)
	return count, err
}

// OPDSSource represents an OPDS catalog source
type OPDSSource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Username    string    `json:"username,omitempty"`
	Password    string    `json:"password,omitempty"`
	IsDefault   bool      `json:"is_default"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateOPDSSource creates a new OPDS source
func (db *DB) CreateOPDSSource(source *OPDSSource) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		INSERT INTO opds_sources (id, name, url, description, username, password, is_default, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, source.ID, source.Name, source.URL, source.Description, source.Username, source.Password,
		source.IsDefault, source.Enabled, source.CreatedAt, source.UpdatedAt)

	return err
}

// UpdateOPDSSource updates an existing OPDS source
func (db *DB) UpdateOPDSSource(source *OPDSSource) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		UPDATE opds_sources SET name = ?, url = ?, description = ?, username = ?, password = ?, is_default = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`, source.Name, source.URL, source.Description, source.Username, source.Password,
		source.IsDefault, source.Enabled, time.Now(), source.ID)

	return err
}

// GetOPDSSource retrieves an OPDS source by ID
func (db *DB) GetOPDSSource(id string) (*OPDSSource, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	source := &OPDSSource{}
	err := db.conn.QueryRow(`
		SELECT id, name, url, description, username, password, is_default, enabled, created_at, updated_at
		FROM opds_sources WHERE id = ?
	`, id).Scan(&source.ID, &source.Name, &source.URL, &source.Description,
		&source.Username, &source.Password, &source.IsDefault, &source.Enabled, &source.CreatedAt, &source.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return source, nil
}

// ListOPDSSources retrieves all OPDS sources
func (db *DB) ListOPDSSources(enabledOnly bool) ([]*OPDSSource, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `SELECT id, name, url, description, username, password, is_default, enabled, created_at, updated_at FROM opds_sources`
	if enabledOnly {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY is_default DESC, name ASC"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*OPDSSource
	for rows.Next() {
		source := &OPDSSource{}
		err := rows.Scan(&source.ID, &source.Name, &source.URL, &source.Description,
			&source.Username, &source.Password, &source.IsDefault, &source.Enabled, &source.CreatedAt, &source.UpdatedAt)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// DeleteOPDSSource deletes an OPDS source by ID
func (db *DB) DeleteOPDSSource(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM opds_sources WHERE id = ?", id)
	return err
}

// InitializeDefaultOPDSSources adds default OPDS sources if none exist
func (db *DB) InitializeDefaultOPDSSources() error {
	sources, err := db.ListOPDSSources(false)
	if err != nil {
		return err
	}

	if len(sources) > 0 {
		return nil // Already have sources
	}

	log.Println("Initializing default OPDS sources")

	defaults := []OPDSSource{
		{
			ID:          "gutenberg",
			Name:        "Project Gutenberg",
			URL:         "https://m.gutenberg.org/ebooks.opds/",
			Description: "Free ebooks from Project Gutenberg. Over 70,000 free ebooks.",
			IsDefault:   true,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "wikisource",
			Name:        "Wikisource",
			URL:         "https://ws-export.wmcloud.org/opds/en/Ready_for_export.xml",
			Description: "Free ebooks from Wikisource. Public domain works ready for export.",
			IsDefault:   true,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "anarchist-library",
			Name:        "The Anarchist Library",
			URL:         "https://theanarchistlibrary.org/opds",
			Description: "Free anarchist texts and books.",
			IsDefault:   true,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "gallica",
			Name:        "Gallica (French)",
			URL:         "https://gallica.bnf.fr/opds",
			Description: "French National Library digital collection. Mostly French language books.",
			IsDefault:   true,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, source := range defaults {
		if err := db.CreateOPDSSource(&source); err != nil {
			logger.Warn("Failed to create default OPDS source %s: %v", source.Name, err)
		}
	}

	return nil
}

// InitializeDefaults initializes the database with default configuration if empty
func (db *DB) InitializeDefaults() error {
	// Check if config is empty (all defaults)
	existingValue, _ := db.GetConfig("server_port")
	if existingValue == "" {
		logger.Info("Initializing database with default configuration")
		cfg := DefaultConfig()
		if err := db.SaveAllConfig(cfg); err != nil {
			return err
		}
	}

	// Initialize default OPDS sources
	if err := db.InitializeDefaultOPDSSources(); err != nil {
		logger.Warn("Failed to initialize OPDS sources: %v", err)
	}

	return nil
}

// ToAppConfig converts storage.Config to the application config format
func (c *Config) ToAppConfig() map[string]interface{} {
	result := map[string]interface{}{
		"log_file":                  c.LogFile,
		"output_dir":                c.OutputDir,
		"temp_dir":                  c.TempDir,
		"default_voice":             c.DefaultVoice,
		"default_provider":          c.DefaultProvider,
		"server_port":               c.ServerPort,
		"server_host":               c.ServerHost,
		"open_browser":              c.OpenBrowser,
		"bit_rate_kbs":              c.BitRateKbs,
		"sample_rate_hz":            c.SampleRateHz,
		"default_speed":             c.DefaultSpeed,
		"default_pitch":             c.DefaultPitch,
		"chapter_gap_seconds":       c.ChapterGapSeconds,
		"pronunciation_dict_file":   c.PronunciationDictFile,
		"use_default_pronunciation": c.UseDefaultPronunciation,
		"max_file_size_mb":          c.MaxFileSizeMB,
		"concurrent_tts_workers":    c.ConcurrentTTSWorkers,
		"concurrent_encoders":       c.ConcurrentEncoders,
		"cloud_api_key":             c.CloudAPIKey,
		"google_tts_endpoint":       c.GoogleTTSEndpoint,
		"azure_tts_endpoint":        c.AzureTTSEndpoint,
		"google_api_key":            c.GoogleAPIKey,
		"opentts_url":               c.OpenTTSURL,
		"rhvoice_url":               c.RHVoiceURL,
		"silero_url":                c.SileroURL,
		"openai_api_key":            c.OpenAIAPIKey,
		"azure_tts_key":             c.AzureTTSKey,
		"azure_tts_region":          c.AzureTTSRegion,
		"audiobookshelf_url":        c.AudiobookshelfURL,
		"audiobookshelf_user":       c.AudiobookshelfUser,
		"audiobookshelf_password":   c.AudiobookshelfPassword,
		"audiobookshelf_library":    c.AudiobookshelfLibrary,
	}
	// Add per-provider TTS workers
	for provider, workers := range c.ProviderTTSWorkers {
		result["tts_workers_"+provider] = workers
	}
	return result
}
