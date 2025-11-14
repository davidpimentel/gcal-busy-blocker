package sync

import (
	"context"
	"fmt"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
)

type CalendarEventsService interface {
	List(calendarId string, startTime string, endTime string, privateProperties map[string]string) ([]*calendar.Event, error)
	Insert(calendarId string, event *calendar.Event) CalendarEventsInsertCall
	Delete(calendarId string, eventId string) CalendarEventsDeleteCall
}

type CalendarEventsInsertCall interface {
	Do(opts ...googleapi.CallOption) (*calendar.Event, error)
}

type CalendarEventsDeleteCall interface {
	Do(opts ...googleapi.CallOption) error
}

// implementation
type calendarEventsService struct {
	service *calendar.Service
}

func (c *calendarEventsService) List(calendarId string, startTime string, endTime string, privateProperties map[string]string) ([]*calendar.Event, error) {
	eventListCall := c.service.Events.List(calendarId).
		SingleEvents(true).
		OrderBy("startTime")

	if startTime != "" {
		eventListCall = eventListCall.TimeMin(startTime)
	}

	if endTime != "" {
		eventListCall = eventListCall.TimeMax(endTime)
	}

	for key, value := range privateProperties {
		eventListCall = eventListCall.PrivateExtendedProperty(fmt.Sprintf("%s=%s", key, value))
	}
	allEvents := []*calendar.Event{}

	err := eventListCall.Pages(context.Background(), func(events *calendar.Events) error {
		allEvents = append(allEvents, events.Items...)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return allEvents, nil
}
func (c *calendarEventsService) Insert(calendarId string, event *calendar.Event) CalendarEventsInsertCall {
	return c.service.Events.Insert(calendarId, event)
}

func (c *calendarEventsService) Delete(calendarId string, eventId string) CalendarEventsDeleteCall {
	return c.service.Events.Delete(calendarId, eventId)
}
