package storage

import (
	"biblio-audiobook-builder-tts/internal/logger"
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
	TempDir         string `json:"temp_dir"`
	DefaultVoice    string `json:"default_voice"`
	DefaultProvider string `json:"default_provider"`

	// Server settings
	ServerPort  string `json:"server_port"`
	ServerHost  string `json:"server_host"`
	OpenBrowser bool   `json:"open_browser"`

	// TTS settings
	DefaultSpeed            float64 `json:"default_speed"`
	DefaultPitch            float64 `json:"default_pitch"`
	ChapterGapSeconds       int     `json:"chapter_gap_seconds"`
	PartGapSeconds          int     `json:"part_gap_seconds"`
	DetectPartSeparators    bool    `json:"detect_part_separators"`
	SentenceBreakMs         int     `json:"sentence_break_ms"`
	ParagraphBreakMs        int     `json:"paragraph_break_ms"`
	ConvertDashesToBreaks   bool    `json:"convert_dashes_to_breaks"`
	DashBreakDurationMs     int     `json:"dash_break_duration_ms"`
	PronunciationDictFile   string  `json:"pronunciation_dict_file"`
	UseDefaultPronunciation bool    `json:"use_default_pronunciation"`
	MaxFileSizeMB           int     `json:"max_file_size_mb"`

	// Performance settings
	ConcurrentEncoders int `json:"concurrent_encoders"`

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
	Language           string           `json:"language"`
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
		language TEXT DEFAULT 'en',
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

	CREATE TABLE IF NOT EXISTS providers (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		enabled BOOLEAN DEFAULT 1,
		url TEXT DEFAULT '',
		api_key TEXT DEFAULT '',
		region TEXT DEFAULT '',
		tts_workers INTEGER DEFAULT 3,
		max_chunk_size INTEGER DEFAULT 900,
		normalize_numbers BOOLEAN DEFAULT 1,
		ssml_support BOOLEAN DEFAULT 0,
		sample_rate INTEGER DEFAULT 48000,
		is_default BOOLEAN DEFAULT 0,
		display_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_providers_enabled ON providers(enabled);
	CREATE INDEX IF NOT EXISTS idx_providers_type ON providers(type);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Add worker_progress and num_workers columns if they don't exist (migration for existing DBs)
	db.conn.Exec("ALTER TABLE jobs ADD COLUMN worker_progress TEXT")
	db.conn.Exec("ALTER TABLE jobs ADD COLUMN num_workers INTEGER DEFAULT 0")

	// Add language column if it doesn't exist (migration for existing DBs)
	db.conn.Exec("ALTER TABLE jobs ADD COLUMN language TEXT DEFAULT 'en'")

	// Add ssml_support column if it doesn't exist (migration for existing DBs)
	db.conn.Exec("ALTER TABLE providers ADD COLUMN ssml_support BOOLEAN DEFAULT 0")

	// Add max_chunk_size column if it doesn't exist (migration for existing DBs)
	db.conn.Exec("ALTER TABLE providers ADD COLUMN max_chunk_size INTEGER DEFAULT 900")

	// Update existing providers with appropriate max_chunk_size values
	db.conn.Exec("UPDATE providers SET max_chunk_size = 5000 WHERE id = 'espeak' AND max_chunk_size = 900")
	db.conn.Exec("UPDATE providers SET max_chunk_size = 4000 WHERE id IN ('google', 'openai', 'azure') AND max_chunk_size = 900")
	db.conn.Exec("UPDATE providers SET max_chunk_size = 2000 WHERE id IN ('opentts', 'rhvoice') AND max_chunk_size = 900")

	// Add sample_rate column if it doesn't exist (migration for existing DBs)
	db.conn.Exec("ALTER TABLE providers ADD COLUMN sample_rate INTEGER DEFAULT 48000")

	// Update existing providers with appropriate sample rates
	db.conn.Exec("UPDATE providers SET sample_rate = 48000 WHERE id = 'silero'")
	db.conn.Exec("UPDATE providers SET sample_rate = 44100 WHERE id = 'openvoice'")
	db.conn.Exec("UPDATE providers SET sample_rate = 44100 WHERE id = 'google'")
	db.conn.Exec("UPDATE providers SET sample_rate = 24000 WHERE id IN ('azure', 'openai', 'rhvoice')")
	db.conn.Exec("UPDATE providers SET sample_rate = 22050 WHERE id IN ('espeak', 'opentts')")

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
	if v, ok := configMap["default_speed"]; ok {
		fmt.Sscanf(v, "%f", &cfg.DefaultSpeed)
	}
	if v, ok := configMap["default_pitch"]; ok {
		fmt.Sscanf(v, "%f", &cfg.DefaultPitch)
	}
	if v, ok := configMap["chapter_gap_seconds"]; ok {
		fmt.Sscanf(v, "%d", &cfg.ChapterGapSeconds)
	}
	if v, ok := configMap["part_gap_seconds"]; ok {
		fmt.Sscanf(v, "%d", &cfg.PartGapSeconds)
	}
	if v, ok := configMap["detect_part_separators"]; ok {
		cfg.DetectPartSeparators = v == "true"
	}
	if v, ok := configMap["sentence_break_ms"]; ok {
		fmt.Sscanf(v, "%d", &cfg.SentenceBreakMs)
	}
	if v, ok := configMap["paragraph_break_ms"]; ok {
		fmt.Sscanf(v, "%d", &cfg.ParagraphBreakMs)
	}
	if v, ok := configMap["convert_dashes_to_breaks"]; ok {
		cfg.ConvertDashesToBreaks = v == "true"
	}
	if v, ok := configMap["dash_break_duration_ms"]; ok {
		fmt.Sscanf(v, "%d", &cfg.DashBreakDurationMs)
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
	if v, ok := configMap["concurrent_encoders"]; ok {
		fmt.Sscanf(v, "%d", &cfg.ConcurrentEncoders)
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
		"temp_dir":                  cfg.TempDir,
		"default_voice":             cfg.DefaultVoice,
		"default_provider":          cfg.DefaultProvider,
		"server_port":               cfg.ServerPort,
		"server_host":               cfg.ServerHost,
		"open_browser":              fmt.Sprintf("%t", cfg.OpenBrowser),
		"default_speed":             fmt.Sprintf("%.2f", cfg.DefaultSpeed),
		"default_pitch":             fmt.Sprintf("%.2f", cfg.DefaultPitch),
		"chapter_gap_seconds":       fmt.Sprintf("%d", cfg.ChapterGapSeconds),
		"pronunciation_dict_file":   cfg.PronunciationDictFile,
		"use_default_pronunciation": fmt.Sprintf("%t", cfg.UseDefaultPronunciation),
		"max_file_size_mb":          fmt.Sprintf("%d", cfg.MaxFileSizeMB),
		"concurrent_encoders":       fmt.Sprintf("%d", cfg.ConcurrentEncoders),
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
		LogFile:                 "biblio-audiobook-builder-tts.log",
		TempDir:                 "./temp",
		DefaultVoice:            "en-US",
		DefaultProvider:         "espeak",
		ServerPort:              "8080",
		ServerHost:              "0.0.0.0",
		OpenBrowser:             true,
		DefaultSpeed:            1.0,
		DefaultPitch:            1.0,
		ChapterGapSeconds:       2,
		PartGapSeconds:          2,
		DetectPartSeparators:    true,
		SentenceBreakMs:         500,
		ParagraphBreakMs:        800,
		ConvertDashesToBreaks:   true,
		DashBreakDurationMs:     300,
		PronunciationDictFile:   "",
		UseDefaultPronunciation: true,
		MaxFileSizeMB:           250,
		ConcurrentEncoders:      2,
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
		INSERT INTO jobs (id, status, file_name, file_path, provider, voice, language, speed, pitch,
			book_title, book_author, output_path, m4b_file, m4b_files, conversion_progress, build_progress,
			current_chapter, total_chapters, current_chapter_num, worker_progress, num_workers, error, created_at, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.Status, job.FileName, job.FilePath, job.Provider, job.Voice, job.Language,
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
		UPDATE jobs SET status = ?, file_name = ?, file_path = ?, provider = ?, voice = ?, language = ?,
			speed = ?, pitch = ?, book_title = ?, book_author = ?, output_path = ?,
			m4b_file = ?, m4b_files = ?, conversion_progress = ?, build_progress = ?, current_chapter = ?,
			total_chapters = ?, current_chapter_num = ?, worker_progress = ?, num_workers = ?, error = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`, job.Status, job.FileName, job.FilePath, job.Provider, job.Voice, job.Language,
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

	var language sql.NullString
	err := db.conn.QueryRow(`
		SELECT id, status, file_name, file_path, provider, voice, language, speed, pitch,
			book_title, book_author, output_path, m4b_file, m4b_files, conversion_progress, build_progress,
			current_chapter, total_chapters, current_chapter_num, worker_progress, num_workers, error, created_at, started_at, completed_at
		FROM jobs WHERE id = ?
	`, id).Scan(&job.ID, &job.Status, &job.FileName, &job.FilePath, &job.Provider,
		&job.Voice, &language, &job.Speed, &job.Pitch, &job.BookTitle, &job.BookAuthor,
		&job.OutputPath, &job.M4BFile, &m4bFilesJSON, &job.ConversionProgress, &job.BuildProgress,
		&job.CurrentChapter, &job.TotalChapters, &job.CurrentChapterNum,
		&workerProgressJSON, &job.NumWorkers, &job.Error, &job.CreatedAt, &startedAt, &completedAt)
	if language.Valid {
		job.Language = language.String
	}

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
		SELECT id, status, file_name, file_path, provider, voice, language, speed, pitch,
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
		var language sql.NullString

		err := rows.Scan(&job.ID, &job.Status, &job.FileName, &job.FilePath, &job.Provider,
			&job.Voice, &language, &job.Speed, &job.Pitch, &job.BookTitle, &job.BookAuthor,
			&job.OutputPath, &job.M4BFile, &m4bFilesJSON, &job.ConversionProgress, &job.BuildProgress,
			&job.CurrentChapter, &job.TotalChapters, &job.CurrentChapterNum,
			&workerProgressJSON, &job.NumWorkers, &job.Error, &job.CreatedAt, &startedAt, &completedAt)
		if err != nil {
			return nil, err
		}

		if language.Valid {
			job.Language = language.String
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
	var language sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, status, file_name, file_path, provider, voice, language, speed, pitch,
			book_title, book_author, output_path, m4b_file, m4b_files, conversion_progress, build_progress,
			current_chapter, total_chapters, current_chapter_num, error, created_at, started_at, completed_at
		FROM jobs WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&job.ID, &job.Status, &job.FileName, &job.FilePath, &job.Provider,
		&job.Voice, &language, &job.Speed, &job.Pitch, &job.BookTitle, &job.BookAuthor,
		&job.OutputPath, &job.M4BFile, &m4bFilesJSON, &job.ConversionProgress, &job.BuildProgress,
		&job.CurrentChapter, &job.TotalChapters, &job.CurrentChapterNum,
		&job.Error, &job.CreatedAt, &startedAt, &completedAt)
	if language.Valid {
		job.Language = language.String
	}

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

// TTSProvider represents a TTS provider configuration stored in the database
type TTSProvider struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"` // "local", "cloud", "self-hosted"
	Enabled          bool      `json:"enabled"`
	URL              string    `json:"url"`
	APIKey           string    `json:"api_key"`
	Region           string    `json:"region"`
	TTSWorkers       int       `json:"tts_workers"`
	MaxChunkSize     int       `json:"max_chunk_size"` // Maximum characters per TTS request
	NormalizeNumbers bool      `json:"normalize_numbers"`
	SSMLSupport      bool      `json:"ssml_support"`
	SampleRate       int       `json:"sample_rate"` // Output sample rate in Hz (e.g., 48000, 44100, 24000)
	IsDefault        bool      `json:"is_default"`
	DisplayOrder     int       `json:"display_order"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreateProvider creates a new provider in the database
func (db *DB) CreateProvider(provider *TTSProvider) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		INSERT INTO providers (id, name, type, enabled, url, api_key, region, tts_workers, max_chunk_size, normalize_numbers, ssml_support, sample_rate, is_default, display_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			enabled = excluded.enabled,
			url = excluded.url,
			api_key = excluded.api_key,
			region = excluded.region,
			tts_workers = excluded.tts_workers,
			max_chunk_size = excluded.max_chunk_size,
			normalize_numbers = excluded.normalize_numbers,
			ssml_support = excluded.ssml_support,
			sample_rate = excluded.sample_rate,
			is_default = excluded.is_default,
			display_order = excluded.display_order,
			updated_at = excluded.updated_at
	`, provider.ID, provider.Name, provider.Type, provider.Enabled, provider.URL, provider.APIKey, provider.Region,
		provider.TTSWorkers, provider.MaxChunkSize, provider.NormalizeNumbers, provider.SSMLSupport, provider.SampleRate, provider.IsDefault, provider.DisplayOrder,
		provider.CreatedAt, provider.UpdatedAt)

	return err
}

// UpdateProvider updates an existing provider
func (db *DB) UpdateProvider(provider *TTSProvider) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		UPDATE providers SET name = ?, type = ?, enabled = ?, url = ?, api_key = ?, region = ?, tts_workers = ?,
			max_chunk_size = ?, normalize_numbers = ?, ssml_support = ?, sample_rate = ?, is_default = ?, display_order = ?, updated_at = ?
		WHERE id = ?
	`, provider.Name, provider.Type, provider.Enabled, provider.URL, provider.APIKey, provider.Region, provider.TTSWorkers,
		provider.MaxChunkSize, provider.NormalizeNumbers, provider.SSMLSupport, provider.SampleRate, provider.IsDefault, provider.DisplayOrder, time.Now(), provider.ID)

	return err
}

// GetProvider retrieves a provider by ID
func (db *DB) GetProvider(id string) (*TTSProvider, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	provider := &TTSProvider{}
	err := db.conn.QueryRow(`
		SELECT id, name, type, enabled, url, api_key, region, tts_workers, max_chunk_size, normalize_numbers, ssml_support, sample_rate, is_default, display_order, created_at, updated_at
		FROM providers WHERE id = ?
	`, id).Scan(&provider.ID, &provider.Name, &provider.Type, &provider.Enabled, &provider.URL, &provider.APIKey, &provider.Region,
		&provider.TTSWorkers, &provider.MaxChunkSize, &provider.NormalizeNumbers, &provider.SSMLSupport, &provider.SampleRate, &provider.IsDefault, &provider.DisplayOrder,
		&provider.CreatedAt, &provider.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return provider, nil
}

// ListProviders retrieves all providers, optionally filtered by enabled status
func (db *DB) ListProviders(enabledOnly bool) ([]*TTSProvider, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `SELECT id, name, type, enabled, url, api_key, region, tts_workers, max_chunk_size, normalize_numbers, ssml_support, sample_rate, is_default, display_order, created_at, updated_at FROM providers`
	if enabledOnly {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY display_order ASC, name ASC"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*TTSProvider
	for rows.Next() {
		provider := &TTSProvider{}
		err := rows.Scan(&provider.ID, &provider.Name, &provider.Type, &provider.Enabled, &provider.URL, &provider.APIKey, &provider.Region,
			&provider.TTSWorkers, &provider.MaxChunkSize, &provider.NormalizeNumbers, &provider.SSMLSupport, &provider.SampleRate, &provider.IsDefault, &provider.DisplayOrder,
			&provider.CreatedAt, &provider.UpdatedAt)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

// DeleteProvider deletes a provider by ID
func (db *DB) DeleteProvider(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM providers WHERE id = ?", id)
	return err
}

// SetDefaultProvider sets a provider as the default (and unsets others)
func (db *DB) SetDefaultProvider(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Unset all defaults
	if _, err := tx.Exec("UPDATE providers SET is_default = 0"); err != nil {
		return err
	}

	// Set the new default
	if _, err := tx.Exec("UPDATE providers SET is_default = 1, updated_at = ? WHERE id = ?", time.Now(), id); err != nil {
		return err
	}

	return tx.Commit()
}

// GetDefaultProvider returns the default provider
func (db *DB) GetDefaultProvider() (*TTSProvider, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	provider := &TTSProvider{}
	err := db.conn.QueryRow(`
		SELECT id, name, type, enabled, url, api_key, region, tts_workers, normalize_numbers, ssml_support, is_default, display_order, created_at, updated_at
		FROM providers WHERE is_default = 1 LIMIT 1
	`).Scan(&provider.ID, &provider.Name, &provider.Type, &provider.Enabled, &provider.URL, &provider.APIKey, &provider.Region,
		&provider.TTSWorkers, &provider.NormalizeNumbers, &provider.SSMLSupport, &provider.IsDefault, &provider.DisplayOrder,
		&provider.CreatedAt, &provider.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return provider, nil
}

// InitializeDefaultProviders adds default TTS providers if none exist
func (db *DB) InitializeDefaultProviders() error {
	providers, err := db.ListProviders(false)
	if err != nil {
		return err
	}

	if len(providers) > 0 {
		return nil // Already have providers
	}

	log.Println("Initializing default TTS providers")

	now := time.Now()
	defaults := []TTSProvider{
		{
			ID:               "espeak",
			Name:             "eSpeak",
			Type:             "local",
			Enabled:          true,
			URL:              "",
			APIKey:           "",
			Region:           "",
			TTSWorkers:       3,
			MaxChunkSize:     5000,
			NormalizeNumbers: true,
			SSMLSupport:      false,
			IsDefault:        true,
			DisplayOrder:     0,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "google",
			Name:             "Google Cloud TTS",
			Type:             "cloud",
			Enabled:          false,
			URL:              "",
			APIKey:           "",
			Region:           "",
			TTSWorkers:       3,
			MaxChunkSize:     4000,
			NormalizeNumbers: false,
			SSMLSupport:      true,
			IsDefault:        false,
			DisplayOrder:     1,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "openai",
			Name:             "OpenAI TTS",
			Type:             "cloud",
			Enabled:          false,
			URL:              "",
			APIKey:           "",
			Region:           "",
			TTSWorkers:       3,
			MaxChunkSize:     4000,
			NormalizeNumbers: false,
			SSMLSupport:      false,
			IsDefault:        false,
			DisplayOrder:     2,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "azure",
			Name:             "Azure TTS",
			Type:             "cloud",
			Enabled:          false,
			URL:              "",
			APIKey:           "",
			Region:           "",
			TTSWorkers:       3,
			MaxChunkSize:     4000,
			NormalizeNumbers: false,
			SSMLSupport:      true,
			IsDefault:        false,
			DisplayOrder:     3,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "opentts",
			Name:             "OpenTTS",
			Type:             "self-hosted",
			Enabled:          false,
			URL:              "",
			APIKey:           "",
			Region:           "",
			TTSWorkers:       3,
			MaxChunkSize:     2000,
			NormalizeNumbers: true,
			SSMLSupport:      false,
			IsDefault:        false,
			DisplayOrder:     4,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "rhvoice",
			Name:             "RHVoice",
			Type:             "self-hosted",
			Enabled:          false,
			URL:              "",
			APIKey:           "",
			Region:           "",
			TTSWorkers:       3,
			MaxChunkSize:     2000,
			NormalizeNumbers: true,
			SSMLSupport:      true,
			IsDefault:        false,
			DisplayOrder:     5,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "silero",
			Name:             "Silero TTS",
			Type:             "self-hosted",
			Enabled:          false,
			URL:              "",
			APIKey:           "",
			Region:           "",
			TTSWorkers:       3,
			MaxChunkSize:     900,
			NormalizeNumbers: true,
			SSMLSupport:      true,
			IsDefault:        false,
			DisplayOrder:     6,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "openvoice",
			Name:             "OpenVoice TTS",
			Type:             "self-hosted",
			Enabled:          false,
			URL:              "",
			APIKey:           "",
			Region:           "",
			TTSWorkers:       3,
			MaxChunkSize:     2000,
			NormalizeNumbers: true,
			SSMLSupport:      false,
			IsDefault:        false,
			DisplayOrder:     7,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}

	for _, provider := range defaults {
		if err := db.CreateProvider(&provider); err != nil {
			logger.Warn("Failed to create default provider %s: %v", provider.Name, err)
		}
	}

	return nil
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

	// Initialize default TTS providers
	if err := db.InitializeDefaultProviders(); err != nil {
		logger.Warn("Failed to initialize TTS providers: %v", err)
	}

	return nil
}

// ToAppConfig converts storage.Config to the application config format
func (c *Config) ToAppConfig() map[string]interface{} {
	return map[string]interface{}{
		"log_file":                  c.LogFile,
		"temp_dir":                  c.TempDir,
		"default_voice":             c.DefaultVoice,
		"default_provider":          c.DefaultProvider,
		"server_port":               c.ServerPort,
		"server_host":               c.ServerHost,
		"open_browser":              c.OpenBrowser,
		"default_speed":             c.DefaultSpeed,
		"default_pitch":             c.DefaultPitch,
		"chapter_gap_seconds":       c.ChapterGapSeconds,
		"part_gap_seconds":          c.PartGapSeconds,
		"detect_part_separators":    c.DetectPartSeparators,
		"sentence_break_ms":         c.SentenceBreakMs,
		"paragraph_break_ms":        c.ParagraphBreakMs,
		"convert_dashes_to_breaks":  c.ConvertDashesToBreaks,
		"dash_break_duration_ms":    c.DashBreakDurationMs,
		"pronunciation_dict_file":   c.PronunciationDictFile,
		"use_default_pronunciation": c.UseDefaultPronunciation,
		"max_file_size_mb":          c.MaxFileSizeMB,
		"concurrent_encoders":       c.ConcurrentEncoders,
		"audiobookshelf_url":        c.AudiobookshelfURL,
		"audiobookshelf_user":       c.AudiobookshelfUser,
		"audiobookshelf_password":   c.AudiobookshelfPassword,
		"audiobookshelf_library":    c.AudiobookshelfLibrary,
	}
}
