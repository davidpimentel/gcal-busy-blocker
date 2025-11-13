package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/davidpimentel/calendar-sync/config"
	"google.golang.org/api/calendar/v3"
)

type SyncClient struct {
	SourceCalendarService      *calendar.Service
	DestinationCalendarService *calendar.Service
	DaysAhead                  int
	DryRun                     bool
}

const (
	defaultCalendar      = "primary"
	propertyAppNameValue = "true"
)

func (s *SyncClient) RunSync() {
	fmt.Println("Starting calendar sync...")

	// Get events from source calendar
	now := time.Now().Format(time.RFC3339)
	endTime := time.Now().AddDate(0, 0, s.DaysAhead).Format(time.RFC3339)

	fmt.Println("Fetching events from source calendar...")
	fmt.Printf("Time range: %s to %s\n", now, endTime)

	// List events from source calendar
	sourceEvents := fetchEvents(s.SourceCalendarService, now, endTime, map[string]string{})

	if len(sourceEvents) == 0 {
		fmt.Println("No upcoming events found in source calendar.")
		return
	}

	fmt.Printf("Found %d events in source calendar\n", len(sourceEvents))

	existingDestinationEvents := fetchEvents(s.DestinationCalendarService, now, endTime, map[string]string{config.AppName: "true"})

	for _, event := range sourceEvents {
		fmt.Printf("Event: %s (%s)\n", event.Summary, event.Id)

		if eventAlreadyExists(existingDestinationEvents, event.Id) {
			fmt.Printf("Event already synced: %s\n", event.Id)
		} else {
			newEvent := createDestinationEvent(event)

			fmt.Printf("Creating new event: %s\n", newEvent.Summary)

			if s.DryRun {
				b, err := json.MarshalIndent(newEvent, "", "  ")
				if err != nil {
					fmt.Println(err)
				}
				fmt.Print(string(b))
			} else {
				_, err := s.DestinationCalendarService.Events.Insert("primary", newEvent).Do()
				if err != nil {
					log.Printf("Error creating event: %v", err)
					continue
				}
			}
		}
	}

	fmt.Println("Sync completed successfully")
}

func fetchEvents(calendarService *calendar.Service, startTime string, endTime string, privateProperies map[string]string) []*calendar.Event {
	eventListCall := calendarService.Events.List(defaultCalendar).
		TimeMin(startTime).
		TimeMax(endTime).
		SingleEvents(true).
		OrderBy("startTime")

	for key, value := range privateProperies {
		eventListCall = eventListCall.PrivateExtendedProperty(fmt.Sprintf("%s=%s", key, value))
	}
	events, err := eventListCall.Do()
	if err != nil {
		log.Fatalf("Unable to retrieve events from source calendar: %v", err)
	}
	return events.Items
}

func createDestinationEvent(sourceEvent *calendar.Event) *calendar.Event {
	return &calendar.Event{
		ColorId:     "4",
		Summary:     "Busy",
		Description: "Created with calendar-sync",
		Start:       sourceEvent.Start,
		End:         sourceEvent.End,
		// Add extended properties to track the source event
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				config.AppName:                  propertyAppNameValue,
				config.SourceEventIdPropertyKey: sourceEvent.Id,
			},
		},
	}
}

func eventAlreadyExists(destinationEvents []*calendar.Event, sourceEventID string) bool {
	for _, event := range destinationEvents {
		if event.ExtendedProperties != nil && event.ExtendedProperties.Private != nil {
			if event.ExtendedProperties.Private[config.SourceEventIdPropertyKey] == sourceEventID {
				return true
			}
		}
	}
	return false
}
