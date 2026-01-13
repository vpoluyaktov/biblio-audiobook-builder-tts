package audio

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FFProbe provides audio file metadata extraction
// Ported from abb_ia project
type FFProbe struct {
	fileName string
	metadata ffprobeMetadata
}

type ffprobeMetadata struct {
	Format struct {
		Filename       string `json:"filename"`
		NbStreams      int    `json:"nb_streams"`
		NbPrograms     int    `json:"nb_programs"`
		FormatName     string `json:"format_name"`
		FormatLongName string `json:"format_long_name"`
		StartTime      string `json:"start_time"`
		Duration       string `json:"duration"`
		Size           string `json:"size"`
		BitRate        string `json:"bit_rate"`
		ProbeScore     int    `json:"probe_score"`
		Tags           struct {
			Title   string `json:"title"`
			Artist  string `json:"artist"`
			Album   string `json:"album"`
			Comment string `json:"comment"`
			Genre   string `json:"genre"`
			Date    string `json:"date"`
		} `json:"tags"`
	} `json:"format"`
}

// NewFFProbe creates a new FFProbe instance for the given file
func NewFFProbe(fileName string) (*FFProbe, error) {
	p := &FFProbe{fileName: fileName}
	args := newArgs().
		appendArgs("-loglevel error").
		appendArgs("-show_format").
		appendArgs("-show_streams").
		appendArgs("-of json").
		appendFileName(p.fileName)
	out, err := exec.Command("ffprobe", args.toSlice()...).Output()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(out, &p.metadata)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Duration returns the duration in seconds
func (p *FFProbe) Duration() float64 {
	f, err := strconv.ParseFloat(p.metadata.Format.Duration, 64)
	if err != nil {
		return 0
	}
	return f
}

// Title returns the title tag or filename if not set
func (p *FFProbe) Title() string {
	if p.metadata.Format.Tags.Title == "" {
		return filepath.Base(p.metadata.Format.Filename)
	}
	return p.metadata.Format.Tags.Title
}

// Size returns the file size in bytes
func (p *FFProbe) Size() int64 {
	s, err := strconv.ParseInt(p.metadata.Format.Size, 0, 64)
	if err != nil {
		return 0
	}
	return s
}

// Format returns the format name with bitrate
func (p *FFProbe) Format() string {
	bitRate, err := strconv.Atoi(p.BitRate())
	if err != nil {
		return p.metadata.Format.FormatName
	}
	return strings.ToUpper(p.metadata.Format.FormatName) + " " + strconv.Itoa(bitRate/1000) + " kb/s"
}

// BitRate returns the bit rate as string
func (p *FFProbe) BitRate() string {
	return p.metadata.Format.BitRate
}

// Artist returns the artist tag
func (p *FFProbe) Artist() string {
	return p.metadata.Format.Tags.Artist
}

// Album returns the album tag
func (p *FFProbe) Album() string {
	return p.metadata.Format.Tags.Album
}
