package audiobookshelf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// Client provides access to the Audiobookshelf API
// Ported from abb_ia project
type Client struct {
	url           string
	userName      string
	password      string
	loginResponse *LoginResponse
}

// NewClient creates a new Audiobookshelf client
func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

// Login authenticates with the Audiobookshelf server
func (c *Client) Login(userName string, password string) error {
	c.userName = userName
	c.password = password

	requestBody := LoginRequest{
		Username: c.userName,
		Password: c.password,
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal login request body: %v", err)
	}

	resp, err := http.Post(c.url+"/login", "application/json", bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return fmt.Errorf("failed to make login API call: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login API call returned status code: %d", resp.StatusCode)
	}

	var loginResp LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return fmt.Errorf("failed to decode login response: %v", err)
	}
	c.loginResponse = &loginResp

	return nil
}

// GetLibraries returns all available libraries
func (c *Client) GetLibraries() ([]Library, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", c.url+"/api/libraries", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.loginResponse.User.Token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var response LibrariesResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body: %v", err)
	}

	return response.Libraries, nil
}

// GetLibraryID returns the ID of a library by name
func (c *Client) GetLibraryID(libraries []Library, libraryName string) (string, error) {
	for _, library := range libraries {
		if library.Name == libraryName {
			return library.ID, nil
		}
	}
	return "", fmt.Errorf("no library with name '%s' found", libraryName)
}

// GetFolders returns the folders for a library by name
func (c *Client) GetFolders(libraries []Library, libraryName string) ([]Folder, error) {
	for _, library := range libraries {
		if library.Name == libraryName {
			return library.Folders, nil
		}
	}
	return nil, fmt.Errorf("no library with name '%s' found", libraryName)
}

// ScanLibrary triggers a library scan
func (c *Client) ScanLibrary(libraryID string) error {
	url := c.url + "/api/libraries/" + libraryID + "/scan"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.loginResponse.User.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusForbidden:
		return fmt.Errorf("an admin user is required to start a scan")
	case http.StatusNotFound:
		return fmt.Errorf("the user cannot access the library or no library with the provided ID exists")
	default:
		return nil
	}
}

// UploadBook uploads an audiobook to the server
type UploadProgressCallback func(fileID int, fileName string, size int64, pos int64, percent int)

// Audiobook represents an audiobook to upload
type Audiobook struct {
	Title  string
	Author string
	Series string
	Files  []string // List of M4B file paths
}

// UploadBook uploads an audiobook to the Audiobookshelf server
func (c *Client) UploadBook(ab *Audiobook, libraryID string, folderID string, callback UploadProgressCallback) error {
	// Open each file for upload
	var filesList []*os.File
	for _, filePath := range ab.Files {
		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()
		filesList = append(filesList, f)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add metadata fields
	_ = writer.WriteField("title", ab.Title)
	_ = writer.WriteField("author", ab.Author)
	_ = writer.WriteField("series", ab.Series)
	_ = writer.WriteField("library", libraryID)
	_ = writer.WriteField("folder", folderID)

	// Add files to the request with progress reporting
	for i, file := range filesList {
		part, err := writer.CreateFormFile(strconv.Itoa(i), filepath.Base(file.Name()))
		if err != nil {
			return err
		}

		// Create a progress reader to track callback
		fileStat, err := file.Stat()
		if err != nil {
			return err
		}
		pr := &progressReader{
			fileID:   i,
			fileName: filepath.Base(file.Name()),
			reader:   file,
			size:     fileStat.Size(),
			callback: callback,
		}

		_, err = io.Copy(part, pr)
		if err != nil {
			return err
		}
	}

	err := writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.url+"/api/upload", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.loginResponse.User.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to upload audiobook: %s", resp.Status)
	}

	return nil
}

// IsLoggedIn returns true if the client is authenticated
func (c *Client) IsLoggedIn() bool {
	return c.loginResponse != nil && c.loginResponse.User.Token != ""
}

// progressReader wraps an io.Reader to report progress
type progressReader struct {
	fileID   int
	fileName string
	reader   io.Reader
	size     int64
	pos      int64
	percent  int
	callback UploadProgressCallback
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if err == nil && pr.callback != nil {
		pr.pos += int64(n)
		pr.percent = int(float64(pr.pos) / float64(pr.size) * 100)
		pr.callback(pr.fileID, pr.fileName, pr.size, pr.pos, pr.percent)
	}
	return n, err
}
