package dto

// DisplayConfigCommand represents a command to display the configuration page
type DisplayConfigCommand struct{}

// SwitchToPageCommand represents a command to switch to a different page
type SwitchToPageCommand struct {
	Name string
}
