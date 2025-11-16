package sync

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/api/calendar/v3"
)

type MockCalendarEventsService struct {
	events         []*calendar.Event
	insertedEvents []*calendar.Event
	deletedEvents  []string
	listCalls      []*listCallParams
}

type listCallParams struct {
	calendarId        string
	startTime         time.Time
	endTime           time.Time
	privateProperties map[string]string
}

func NewMockCalendarEventsService(events []*calendar.Event) *MockCalendarEventsService {
	return &MockCalendarEventsService{
		events:         events,
		insertedEvents: []*calendar.Event{},
		deletedEvents:  []string{},
		listCalls:      []*listCallParams{},
	}
}

func (m *MockCalendarEventsService) List(calendarId string, startTime time.Time, endTime time.Time, privateProperties map[string]string) ([]*calendar.Event, error) {
	m.listCalls = append(m.listCalls, &listCallParams{
		calendarId:        calendarId,
		startTime:         startTime,
		endTime:           endTime,
		privateProperties: privateProperties,
	})

	return m.events, nil
}

func (m *MockCalendarEventsService) Insert(calendarId string, event *calendar.Event) (*calendar.Event, error) {
	m.insertedEvents = append(m.insertedEvents, event)
	return event, nil
}

func (m *MockCalendarEventsService) Delete(calendarId string, eventId string) error {
	m.deletedEvents = append(m.deletedEvents, eventId)
	return nil
}

// Helper function to create a test event
func createTestEvent(id string, summary string, startTime, endTime time.Time, privateProps map[string]string) *calendar.Event {
	event := &calendar.Event{
		Id:      id,
		Summary: summary,
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
		},
	}

	if privateProps != nil {
		event.ExtendedProperties = &calendar.EventExtendedProperties{
			Private: privateProps,
		}
	}

	return event
}

func eventTimesEqual(event *calendar.Event, event2 *calendar.Event) bool {
	return event.Start.DateTime != event2.Start.DateTime || event.End.DateTime != event2.End.DateTime
}

func TestRunSync(t *testing.T) {
	mockSourceService := &MockCalendarEventsService{
		events: []*calendar.Event{
			createTestEvent("123", "test summary", time.Now(), time.Now().Add(time.Hour), nil),
			createTestEvent("456", "test summary2", time.Now(), time.Now(), nil),
		}}
	mockDestinationService := &MockCalendarEventsService{}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.RunSync(30, false)

	if len(mockSourceService.listCalls) != 1 {
		t.Errorf("Sync should only call source List once")
	}
	if len(mockDestinationService.listCalls) != 1 {
		t.Errorf("Sync should only call destination List once")
	}

	for i, event := range mockDestinationService.insertedEvents {
		sourceEvent := mockSourceService.events[i]
		if eventTimesEqual(event, sourceEvent) {
			t.Errorf("start and end times are not the same between source and destination events")
		}

		if event.ExtendedProperties.Private[appName] != propertyAppNameValue {
			t.Errorf("appName private property not set for destination event")
		}

		if event.ExtendedProperties.Private[sourceEventIdPropertyKey] != sourceEvent.Id {
			t.Errorf("Source Event ID private property not set")
		}
	}
}

func TestRunSyncAlreadyAdded(t *testing.T) {
	mockEvents := []*calendar.Event{
		createTestEvent("123", "test summary", time.Now(), time.Now().Add(time.Hour), nil),
		createTestEvent("456", "test summary2", time.Now(), time.Now(), nil),
	}
	mockDestinationEvents := []*calendar.Event{}
	for _, event := range mockEvents {
		mockDestinationEvents = append(
			mockDestinationEvents,
			createTestEvent("abc", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: event.Id}),
		)
	}

	mockSourceService := &MockCalendarEventsService{
		events: mockEvents}
	mockDestinationService := &MockCalendarEventsService{events: mockDestinationEvents}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.RunSync(30, false)

	if len(mockDestinationService.insertedEvents) != 0 {
		t.Error("An event was inserted when it shouldn't be")
	}
}

func TestRunSyncDeleteOldEvents(t *testing.T) {
	mockEvents := []*calendar.Event{
		createTestEvent("123", "test summary", time.Now(), time.Now().Add(time.Hour), nil),
		createTestEvent("456", "test summary2", time.Now(), time.Now(), nil),
	}
	mockDestinationEvents := []*calendar.Event{
		createTestEvent("123", "Busy", time.Now().AddDate(0, 0, -1), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: "old key"}),
		createTestEvent("abc", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: mockEvents[0].Id}),
		createTestEvent("def", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: "oldKey"}),
	}

	mockSourceService := &MockCalendarEventsService{
		events: mockEvents}
	mockDestinationService := &MockCalendarEventsService{events: mockDestinationEvents}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.RunSync(30, false)

	if len(mockDestinationService.deletedEvents) != 2 || mockDestinationService.deletedEvents[0] != "123" || mockDestinationService.deletedEvents[1] != "def" {
		t.Error("Did not delete old event")
	}

	if !mockDestinationService.listCalls[0].startTime.IsZero() {
		t.Error("Should pass a zero time for start time to get all events")
	}
}

func TestSyncDoesntDeleteOtherEvents(t *testing.T) {
	mockEvents := []*calendar.Event{
		createTestEvent("123", "test summary", time.Now(), time.Now().Add(time.Hour), nil),
		createTestEvent("456", "test summary2", time.Now(), time.Now(), nil),
	}
	mockDestinationEvents := []*calendar.Event{
		createTestEvent("abc", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: mockEvents[0].Id}),
		createTestEvent("def", "Busy", time.Now(), time.Now(), map[string]string{sourceEventIdPropertyKey: "oldKey"}),
	}

	mockSourceService := &MockCalendarEventsService{
		events: mockEvents}
	mockDestinationService := &MockCalendarEventsService{events: mockDestinationEvents}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.RunSync(30, false)

	if len(mockDestinationService.deletedEvents) != 0 {
		t.Error("Deleted an event it shouldn't")
	}
}

func TestRunSyncDryRun(t *testing.T) {
	mockSourceService := &MockCalendarEventsService{events: []*calendar.Event{createTestEvent("123", "test summary", time.Now(), time.Now(), nil)}}
	mockDestinationService := &MockCalendarEventsService{}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.RunSync(30, true)

	if len(mockDestinationService.insertedEvents) != 0 {
		t.Errorf("Expected dry run to insert 0 events, inserted %d", len(mockDestinationService.insertedEvents))
	}
}

func TestClean(t *testing.T) {
	mockDestinationEvents := []*calendar.Event{
		createTestEvent("abc", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: "456"}),
		createTestEvent("123", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: "def"}),
	}

	mockSourceService := &MockCalendarEventsService{}
	mockDestinationService := &MockCalendarEventsService{events: mockDestinationEvents}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	err := syncClient.Clean(false)
	if err != nil {
		t.Errorf("Function returned error: %v", err)
	}

	if len(mockDestinationService.listCalls) != 1 {
		t.Error("Didn't fetch destination events")
	}

	listCall := mockDestinationService.listCalls[len(mockDestinationService.listCalls)-1]
	if listCall.privateProperties[appName] != propertyAppNameValue {
		t.Error("Didn't filter by appName private property")
	}

	if len(mockDestinationService.deletedEvents) != 2 {
		t.Error("Didn't delete all events")
	}
}
func TestCleanDryRun(t *testing.T) {
	mockDestinationEvents := []*calendar.Event{
		createTestEvent("abc", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: "456"}),
		createTestEvent("123", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: "def"}),
	}

	mockSourceService := &MockCalendarEventsService{}
	mockDestinationService := &MockCalendarEventsService{events: mockDestinationEvents}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.Clean(true)

	if len(mockDestinationService.deletedEvents) != 0 {
		t.Error("Delete shouldn't be called during a dry run")
	}
}
func TestCleanDoesntDeleteOtherEvents(t *testing.T) {
	mockDestinationEvents := []*calendar.Event{
		createTestEvent("abc", "Busy", time.Now(), time.Now(), map[string]string{sourceEventIdPropertyKey: "456"}),
		createTestEvent("123", "Busy", time.Now(), time.Now(), map[string]string{}),
	}

	mockSourceService := &MockCalendarEventsService{}
	mockDestinationService := &MockCalendarEventsService{events: mockDestinationEvents}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	err := syncClient.Clean(false)

	if err == nil {
		t.Error("function should have returned an error")
		return
	}

	if !strings.HasSuffix(err.Error(), "abc") {
		t.Error("error returned wasn't related to the first event")
	}

	if len(mockDestinationService.deletedEvents) > 0 {
		t.Error("Delete was called")
	}
}

func TestDeleteDestinationEvent(t *testing.T) {

	mockSourceService := &MockCalendarEventsService{}
	mockDestinationService := &MockCalendarEventsService{}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}
	event := createTestEvent("123", "summary", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: "key"})

	syncClient.deleteDestinationEvent(event, false)

	if len(mockDestinationService.deletedEvents) != 1 {
		t.Error("did not delete event")
	}
}

func TestDeleteDestinationEventDoesntDeleteOtherEvents(t *testing.T) {

	mockSourceService := &MockCalendarEventsService{}
	mockDestinationService := &MockCalendarEventsService{}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}
	event := createTestEvent("123", "summary", time.Now(), time.Now(), map[string]string{})

	syncClient.deleteDestinationEvent(event, false)

	if len(mockDestinationService.deletedEvents) != 0 {
		t.Error("Deleted an event it shouldn't")
	}
}

func TestDeleteDestinationEventDryRun(t *testing.T) {

	mockSourceService := &MockCalendarEventsService{}
	mockDestinationService := &MockCalendarEventsService{}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}
	event := createTestEvent("123", "summary", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: "key"})

	syncClient.deleteDestinationEvent(event, true)

	if len(mockDestinationService.deletedEvents) != 0 {
		t.Error("deleted event when it shouldn't")
	}
}
