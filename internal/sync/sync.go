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
	SourceCalendarService      CalendarEventsService
	DestinationCalendarService CalendarEventsService
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
		SourceCalendarService:      &calendarEventsService{service: sourceSrv},
		DestinationCalendarService: &calendarEventsService{service: destSrv},
	}
}

func (s *SyncClient) RunSync(daysAhead int, dryRun bool) error {
	if dryRun {
		log.Println("DRY RUN!")
	}

	now := time.Now()
	endTime := now.AddDate(0, 0, daysAhead)

	log.Printf("Starting calendar sync for time range: %s to %s\n", now, endTime)

	// List events from source calendar
	sourceEvents := s.fetchSourceEvents(now, endTime)

	if len(sourceEvents) == 0 {
		log.Println("No upcoming events found in source calendar, terminating...")
		return nil
	}

	sourceEventCount := len(sourceEvents)
	skippedEvents := 0
	eventsCreated := 0
	deletedEvents := 0

	existingDestinationEvents := s.fetchBusyBlockEvents(endTime)

	for _, event := range sourceEvents {
		if eventAlreadyExists(existingDestinationEvents, event) {
			skippedEvents++
		} else {
			eventsCreated++
			newEvent := createDestinationEvent(event)

			if dryRun {
				b, err := json.MarshalIndent(newEvent, "", "  ")
				if err != nil {
					return err
				}
				log.Println("Dry Run:")
				log.Println(string(b))
			} else {
				_, err := s.DestinationCalendarService.Insert("primary", newEvent)
				if err != nil {
					log.Printf("Error creating event: %v", err)
					return err
				}
			}
		}
	}

	// Remove blocks that don't exist in source calendar anymore, or are in the past
	oldEvents := findOldEvents(sourceEvents, existingDestinationEvents)
	for _, event := range oldEvents {
		deletedEvents++
		err := s.deleteDestinationEvent(event, dryRun)
		if err != nil {
			return err
		}
	}

	log.Println("Sync completed successfully")
	log.Printf(
		"Source events scanned: %d\nEvents skipped: %d\nEvents added: %d\nEvents deleted: %d",
		sourceEventCount,
		skippedEvents,
		eventsCreated,
		deletedEvents,
	)
	return nil
}

func (s *SyncClient) fetchBusyBlockEvents(endTime time.Time) []*calendar.Event {
	events, err := s.DestinationCalendarService.List(defaultCalendar, time.Time{}, endTime, map[string]string{appName: propertyAppNameValue})
	if err != nil {
		log.Fatalf("Unable to fetch destination calendar events: %v", err)
	}
	return events
}

func (s *SyncClient) fetchSourceEvents(startTime time.Time, endTime time.Time) []*calendar.Event {
	events, err := s.SourceCalendarService.List(defaultCalendar, startTime, endTime, nil)
	if err != nil {
		log.Fatalf("Unable to fetch source calendar events: %v", err)
	}
	return events
}

func findOldEvents(sourceEvents []*calendar.Event, destinationEvents []*calendar.Event) []*calendar.Event {
	sourceEventLookup := map[string]*calendar.Event{}
	for _, event := range sourceEvents {
		sourceEventLookup[event.Id] = event
	}

	oldEvents := []*calendar.Event{}

	for _, destinationEvent := range destinationEvents {
		if sourceEvent := sourceEventLookup[destinationEvent.ExtendedProperties.Private[sourceEventIdPropertyKey]]; sourceEvent == nil ||
			!startAndEndTimesEqual(sourceEvent, destinationEvent) {
			oldEvents = append(oldEvents, destinationEvent)
		}
	}
	return oldEvents
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

func eventAlreadyExists(destinationEvents []*calendar.Event, sourceEvent *calendar.Event) bool {
	for _, event := range destinationEvents {
		if event.ExtendedProperties != nil && event.ExtendedProperties.Private != nil {
			if event.ExtendedProperties.Private[sourceEventIdPropertyKey] == sourceEvent.Id &&
				startAndEndTimesEqual(sourceEvent, event) {
				return true
			}
		}
	}
	return false
}

func (s *SyncClient) Clean(dryRun bool) error {
	events := s.fetchBusyBlockEvents(time.Time{})

	for _, event := range events {
		err := s.deleteDestinationEvent(event, dryRun)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SyncClient) deleteDestinationEvent(event *calendar.Event, dryRun bool) error {
	// Sanity check, ensure each event is definitely ours
	if event.ExtendedProperties.Private[appName] != propertyAppNameValue {
		return fmt.Errorf("aborting, almost deleted an event we weren't supposed to! Event ID = %s", event.Id)
	}

	if dryRun {
		fmt.Printf("DRY RUN - Deleting event at %s - %s\n", event.Start.DateTime, event.End.DateTime)
	} else {
		fmt.Printf("Deleting event at %s - %s\n", event.Start.DateTime, event.End.DateTime)

		err := s.DestinationCalendarService.Delete(defaultCalendar, event.Id)
		if err != nil {
			return fmt.Errorf("error deleting event %s: %v", event.Id, err)
		}
	}
	return nil
}

func startAndEndTimesEqual(eventA *calendar.Event, eventB *calendar.Event) bool {
	return eventA.Start.DateTime == eventB.Start.DateTime && eventA.End.DateTime == eventB.End.DateTime
}
