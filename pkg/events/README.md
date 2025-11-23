# Event System Package

Laravel-inspired event-driven architecture for Conduit-Go.

## Features

- **Loose Coupling**: Decouple business logic with events
- **Multiple Listeners**: Attach multiple listeners to one event
- **Async Support**: Run listeners in background goroutines
- **Thread-Safe**: Safe for concurrent use
- **Conditional Listeners**: Run listeners based on conditions
- **Wildcard Support**: Listen to multiple events at once

## Quick Start

### 1. Create Dispatcher

```go
import (
    "github.com/biyonik/conduit-go/pkg/events"
    "log"
)

dispatcher := events.NewDispatcher(log.Default())
```

### 2. Register Listeners

```go
// Simple function listener
dispatcher.Listen("user.registered", events.ListenerFunc(func(e events.Event) error {
    user := e.Payload().(*models.User)
    log.Printf("New user: %s", user.Email)
    return nil
}))

// Struct listener
type SendWelcomeEmail struct {
    mailer Mailer
}

func (l *SendWelcomeEmail) Handle(event events.Event) error {
    user := event.Payload().(*models.User)
    return l.mailer.Send(user.Email, "Welcome to Conduit!")
}

dispatcher.Listen("user.registered", &SendWelcomeEmail{mailer})
```

### 3. Dispatch Events

```go
// Create and dispatch event
event := events.NewUserRegisteredEvent(user)
dispatcher.Dispatch(event)

// Async dispatch (non-blocking)
dispatcher.DispatchAsync(event)
```

## Advanced Usage

### Async Listeners

Run slow operations in background:

```go
slowListener := &SendEmailListener{}
asyncListener := events.NewAsyncListener(slowListener, logger)
dispatcher.Listen("user.registered", asyncListener)
```

### Conditional Listeners

Run listeners only when condition is met:

```go
listener := &SendEmailListener{}
condition := func(e events.Event) bool {
    user := e.Payload().(*models.User)
    return user.EmailVerified
}
conditionalListener := events.NewConditionalListener(listener, condition)
dispatcher.Listen("user.registered", conditionalListener)
```

### Subscribe to Multiple Events

```go
logListener := &LogActivityListener{}
dispatcher.Subscribe(
    []string{"user.created", "user.updated", "user.deleted"},
    logListener,
)
```

## Built-in Events

```go
// User events
events.EventUserRegistered
events.EventUserUpdated
events.EventUserDeleted
events.EventUserLoggedIn
events.EventUserLoggedOut

// Email events
events.EventEmailSent
events.EventEmailFailed

// Payment events
events.EventPaymentReceived
events.EventPaymentFailed

// ...and more (see event.go)
```

## Custom Events

```go
type OrderPlaced struct {
    events.BaseEvent
    Order *Order
}

func NewOrderPlaced(order *Order) *OrderPlaced {
    return &OrderPlaced{
        BaseEvent: *events.NewBaseEvent("order.placed", order),
        Order:     order,
    }
}

// Usage
event := NewOrderPlaced(order)
dispatcher.Dispatch(event)
```

## Best Practices

1. **Keep Listeners Small**: Each listener should do one thing
2. **Use Async for Slow Operations**: Email, API calls, etc.
3. **Handle Errors Gracefully**: Don't let one listener break others
4. **Use Type Assertions Safely**: Always check payload types
5. **Document Event Payloads**: Make it clear what data events carry

## Testing

```go
func TestUserRegistration(t *testing.T) {
    dispatcher := events.NewDispatcher(log.Default())

    var eventReceived bool
    dispatcher.Listen("user.registered", events.ListenerFunc(func(e events.Event) error {
        eventReceived = true
        return nil
    }))

    event := events.NewUserRegisteredEvent(user)
    dispatcher.Dispatch(event)

    assert.True(t, eventReceived)
}
```

## Integration with Conduit-Go

Add to your application bootstrap:

```go
// cmd/api/main.go
dispatcher := events.NewDispatcher(logger)

// Register listeners
dispatcher.Listen(events.EventUserRegistered, &SendWelcomeEmail{mailer})
dispatcher.Listen(events.EventUserRegistered, &UpdateUserStats{db})

// Inject into container
container.Singleton("events.dispatcher", dispatcher)

// In controllers
dispatcher := container.Get("events.dispatcher").(*events.Dispatcher)
event := events.NewUserRegisteredEvent(user)
dispatcher.DispatchAsync(event) // Non-blocking
```
