package audiobookshelf

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User                 User           `json:"user"`
	UserDefaultLibraryID string         `json:"userDefaultLibraryId"`
	ServerSettings       ServerSettings `json:"serverSettings"`
	Source               string         `json:"Source"`
}

// User represents a user in the system
type User struct {
	ID                  string        `json:"id"`
	Username            string        `json:"username"`
	Type                string        `json:"type"`
	Token               string        `json:"token"`
	IsActive            bool          `json:"isActive"`
	IsLocked            bool          `json:"isLocked"`
	LastSeen            int64         `json:"lastSeen"`
	CreatedAt           int64         `json:"createdAt"`
	Permissions         Permissions   `json:"permissions"`
	LibrariesAccessible []interface{} `json:"librariesAccessible"`
	ItemTagsAccessible  []interface{} `json:"itemTagsAccessible"`
}

// Permissions represents user permissions
type Permissions struct {
	Download              bool `json:"download"`
	Update                bool `json:"update"`
	Delete                bool `json:"delete"`
	Upload                bool `json:"upload"`
	AccessAllLibraries    bool `json:"accessAllLibraries"`
	AccessAllTags         bool `json:"accessAllTags"`
	AccessExplicitContent bool `json:"accessExplicitContent"`
}

// ServerSettings represents server configuration
type ServerSettings struct {
	ID                    string `json:"id"`
	ScannerFindCovers     bool   `json:"scannerFindCovers"`
	StoreCoverWithItem    bool   `json:"storeCoverWithItem"`
	StoreMetadataWithItem bool   `json:"storeMetadataWithItem"`
	MetadataFileFormat    string `json:"metadataFileFormat"`
	Language              string `json:"language"`
	Version               string `json:"version"`
}

// LibrariesResponse represents the response from the libraries endpoint
type LibrariesResponse struct {
	Libraries []Library `json:"libraries"`
}

// Library represents an Audiobookshelf library
type Library struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Folders      []Folder        `json:"folders"`
	DisplayOrder int             `json:"displayOrder"`
	Icon         string          `json:"icon"`
	MediaType    string          `json:"mediaType"`
	Provider     string          `json:"provider"`
	Settings     LibrarySettings `json:"settings"`
	CreatedAt    int64           `json:"createdAt"`
	LastUpdate   int64           `json:"lastUpdate"`
}

// Folder represents a folder in a library
type Folder struct {
	ID        string `json:"id"`
	FullPath  string `json:"fullPath"`
	LibraryID string `json:"libraryId"`
	AddedAt   int64  `json:"addedAt,omitempty"`
}

// LibrarySettings represents library-specific settings
type LibrarySettings struct {
	CoverAspectRatio          float64 `json:"coverAspectRatio"`
	DisableWatcher            bool    `json:"disableWatcher"`
	SkipMatchingMediaWithASIN bool    `json:"skipMatchingMediaWithAsin"`
	SkipMatchingMediaWithISBN bool    `json:"skipMatchingMediaWithIsbn"`
	AutoScanCronExpression    string  `json:"autoScanCronExpression,omitempty"`
}
