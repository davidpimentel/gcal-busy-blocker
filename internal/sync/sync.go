package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/davidpimentel/gcal-busy-blocker/internal/auth"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type SyncClient struct {
	SourceCalendarService      *calendar.Service
	DestinationCalendarService *calendar.Service
}

const (
	appName                  = "gcal-busy-blocker"
	defaultCalendar          = "primary"
	propertyAppNameValue     = "true"
	sourceEventIdPropertyKey = "gcal-busy-blocker-source-event-id"
)

func NewSyncClient() *SyncClient {

	// Get source client
	sourceClient, err := auth.SourceClient()
	if err != nil {
		log.Fatalf("Unable to get source client: %v", err)
	}

	// Get destination client
	destClient, err := auth.DestinationClient()
	if err != nil {
		log.Fatalf("Unable to get destination client: %v", err)
	}

	// Create calendar service for source and destination
	sourceSrv, err := calendar.NewService(context.Background(), option.WithHTTPClient(sourceClient))
	if err != nil {
		log.Fatalf("Unable to retrieve source Calendar client: %v", err)
	}

	destSrv, err := calendar.NewService(context.Background(), option.WithHTTPClient(destClient))
	if err != nil {
		log.Fatalf("Unable to retrieve destination Calendar client: %v", err)
	}

	return &SyncClient{
		SourceCalendarService:      sourceSrv,
		DestinationCalendarService: destSrv,
	}
}

func (s *SyncClient) RunSync(daysAhead int, dryRun bool) {
	if dryRun {
		fmt.Println("DRY RUN!")
	}
	fmt.Println("Starting calendar sync...")

	// Get events from source calendar
	now := time.Now().Format(time.RFC3339)
	endTime := time.Now().AddDate(0, 0, daysAhead).Format(time.RFC3339)

	fmt.Println("Fetching events from source calendar...")
	fmt.Printf("Time range: %s to %s\n", now, endTime)

	// List events from source calendar
	sourceEvents := s.fetchSourceEvents(now, endTime)

	if len(sourceEvents) == 0 {
		fmt.Println("No upcoming events found in source calendar.")
		return
	}

	fmt.Printf("Found %d events in source calendar\n", len(sourceEvents))

	existingDestinationEvents := s.fetchBusyBlockEvents(now, endTime)

	for _, event := range sourceEvents {
		fmt.Printf("Event: %s (%s)\n", event.Summary, event.Id)

		if eventAlreadyExists(existingDestinationEvents, event.Id) {
			fmt.Printf("Event already synced: %s\n", event.Id)
		} else {
			newEvent := createDestinationEvent(event)

			fmt.Println("Creating new event")

			if dryRun {
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

func (s *SyncClient) fetchBusyBlockEvents(startTime string, endTime string) []*calendar.Event {
	return fetchEvents(s.DestinationCalendarService, startTime, endTime, map[string]string{appName: propertyAppNameValue})
}

func (s *SyncClient) fetchSourceEvents(startTime string, endTime string) []*calendar.Event {
	return fetchEvents(s.SourceCalendarService, startTime, endTime, nil)
}

func fetchEvents(calendarService *calendar.Service, startTime string, endTime string, privateProperies map[string]string) []*calendar.Event {
	eventListCall := calendarService.Events.List(defaultCalendar).
		SingleEvents(true).
		OrderBy("startTime")

	if startTime != "" {
		eventListCall = eventListCall.TimeMin(startTime)
	}

	if endTime != "" {
		eventListCall = eventListCall.TimeMax(endTime)
	}

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
		Description: "Created with <a href=\"https://github.com/davidpimentel/gcal-busy-blocker\">gcal-busy-blocker</a>. User has a personal commitment and is busy at this time. Please find another time to avoid scheduling conflicts.",
		Start:       sourceEvent.Start,
		End:         sourceEvent.End,
		// Add extended properties to track the source event
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				appName:                  propertyAppNameValue,
				sourceEventIdPropertyKey: sourceEvent.Id,
			},
		},
		Source: &calendar.EventSource{
			Title: "gcal-busy-blocker",
			Url:   "https://github.com/davidpimentel/gcal-busy-blocker",
		},
	}
}

func eventAlreadyExists(destinationEvents []*calendar.Event, sourceEventID string) bool {
	for _, event := range destinationEvents {
		if event.ExtendedProperties != nil && event.ExtendedProperties.Private != nil {
			if event.ExtendedProperties.Private[sourceEventIdPropertyKey] == sourceEventID {
				return true
			}
		}
	}
	return false
}

func (s *SyncClient) Clean(dryRun bool) {
	events := s.fetchBusyBlockEvents("", "")

	for _, event := range events {
		if dryRun {
			fmt.Printf("DRY RUN - Deleting event at %s - %s\n", event.Start.DateTime, event.End.DateTime)
		} else {
			fmt.Printf("Deleting event at %s - %s\n", event.Start.DateTime, event.End.DateTime)
			err := s.DestinationCalendarService.Events.Delete(defaultCalendar, event.Id).Do()
			if err != nil {
				log.Fatalf("Error deleting event %s: %v", event.Id, err)
			}
		}
	}
}
