// config/config.go

package config

import "google.golang.org/api/calendar/v3"

// Application constants
const (
	// Application name
	AppName = "calendar-sync"

	// File paths
	CredentialsFile = "credentials.json"
	SourceTokenFile = "source_token.json"
	DestTokenFile   = "destination_token.json"

	// Extended property keys for tracking events
	SourceEventIdPropertyKey  = "calendar-sync-source-event-id"
	SourceCalendarPropertyKey = "calendar-sync-source-calendar-id"
)

// Google Calendar permission scopes
var (
	SourceScope      = []string{calendar.CalendarReadonlyScope, calendar.CalendarEventsReadonlyScope}
	DestinationScope = []string{calendar.CalendarScope}
)

// TimeConfig contains time-related configuration
type TimeConfig struct {
	// Number of days to look ahead for events
	DaysAhead int
}

// DefaultTimeConfig returns the default time configuration
func DefaultTimeConfig() TimeConfig {
	return TimeConfig{
		DaysAhead: 30, // Default to one month
	}
}

// EventConfig contains event-related configuration
type EventConfig struct {
	// Color ID for created events
	ColorID string
	// Summary for created events
	Summary string
	// Description for created events
	Description string
}

// DefaultEventConfig returns the default event configuration
func DefaultEventConfig() EventConfig {
	return EventConfig{
		ColorID:     "4", // Default color (blue)
		Summary:     "Busy",
		Description: "Created with calendar-sync",
	}
}
