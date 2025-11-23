# Mail Package

Laravel-inspired email system for Conduit-Go with SMTP support.

## Features

- **Fluent Message Builder**: Chain methods for easy email construction
- **SMTP Driver**: Send emails via any SMTP server
- **HTML & Plain Text**: Support for both formats
- **Attachments**: Add files to emails
- **Multiple Recipients**: To, Cc, Bcc support
- **Priority Levels**: High, Normal, Low priority
- **Custom Headers**: Add custom email headers
- **Log Driver**: Development/testing without sending real emails

## Quick Start

### 1. Configure SMTP

```go
import "github.com/biyonik/conduit-go/pkg/mail"

// Gmail example
config := &mail.SMTPConfig{
    Host:     "smtp.gmail.com",
    Port:     587,
    Username: "your@gmail.com",
    Password: "app-password",
    From:     mail.Address{Email: "noreply@app.com", Name: "My App"},
    UseTLS:   true,
}

mailer := mail.NewSMTPMailer(config, logger)
```

### 2. Send Email

```go
message := mail.NewMessage().
    To("user@example.com", "John Doe").
    Subject("Welcome to Conduit!").
    Body("Thank you for joining our platform.").
    Html("<h1>Welcome!</h1><p>Thank you for joining.</p>")

err := mailer.Send(message)
if err != nil {
    log.Printf("Failed to send email: %v", err)
}
```

## SMTP Configurations

### Mailhog (Development)

```go
config := &mail.SMTPConfig{
    Host: "localhost",
    Port: 1025,
    From: mail.Address{Email: "dev@conduit.local", Name: "Conduit Dev"},
}
```

### Gmail

```go
config := &mail.SMTPConfig{
    Host:     "smtp.gmail.com",
    Port:     587,
    Username: "your@gmail.com",
    Password: "app-password", // Use App Password, not regular password
    UseTLS:   true,
}
```

### SendGrid

```go
config := &mail.SMTPConfig{
    Host:     "smtp.sendgrid.net",
    Port:     587,
    Username: "apikey",
    Password: "your-sendgrid-api-key",
    UseTLS:   true,
}
```

### AWS SES

```go
config := &mail.SMTPConfig{
    Host:     "email-smtp.us-east-1.amazonaws.com",
    Port:     587,
    Username: "your-smtp-username",
    Password: "your-smtp-password",
    UseTLS:   true,
}
```

## Advanced Usage

### Multiple Recipients

```go
message := mail.NewMessage().
    To("user1@example.com", "User One").
    To("user2@example.com", "User Two").
    Cc("manager@example.com", "Manager").
    Bcc("admin@example.com", "") // No name
```

### Attachments

```go
message := mail.NewMessage().
    To("user@example.com").
    Subject("Your Invoice").
    Body("Please find attached invoice.").
    Attach("/path/to/invoice.pdf").
    Attach("/path/to/report.xlsx")
```

### Priority

```go
message := mail.NewMessage().
    To("admin@example.com").
    Subject("URGENT: Server Down").
    Body("Server is experiencing issues").
    Priority(mail.PriorityHigh)
```

### Custom Headers

```go
message := mail.NewMessage().
    To("user@example.com").
    Subject("Newsletter").
    Body("...").
    Header("X-Campaign-ID", "summer-2024").
    Header("X-Unsubscribe", "https://app.com/unsubscribe")
```

### Reply-To

```go
message := mail.NewMessage().
    From("noreply@app.com", "My App").
    To("user@example.com").
    ReplyTo("support@app.com", "Support Team").
    Subject("Thank you for contacting us")
```

## Log Driver (Development)

For development/testing, use LogMailer to see emails in logs without sending:

```go
mailer := mail.NewLogMailer(logger)

message := mail.NewMessage().
    To("test@example.com").
    Subject("Test Email").
    Body("This won't be sent, just logged")

mailer.Send(message) // Logs email instead of sending
```

## Integration with Queue System

```go
// In SendEmailJob
func (j *SendEmailJob) Handle() error {
    mailer := j.container.Get("mailer").(mail.Mailer)

    message := mail.NewMessage().
        To(j.To).
        Subject(j.Subject).
        Body(j.Body)

    return mailer.Send(message)
}
```

## Integration with Events

```go
// Listen to user registration event
dispatcher.Listen(events.EventUserRegistered, events.ListenerFunc(func(e events.Event) error {
    user := e.Payload().(*models.User)

    message := mail.NewMessage().
        To(user.Email, user.Name).
        Subject("Welcome!").
        Html(renderTemplate("welcome", user))

    return mailer.Send(message)
}))
```

## Error Handling

```go
message := mail.NewMessage().
    To("user@example.com").
    Subject("Test")

err := mailer.Send(message)
if err != nil {
    // Handle errors
    switch {
    case strings.Contains(err.Error(), "validation failed"):
        log.Println("Invalid message")
    case strings.Contains(err.Error(), "smtp send failed"):
        log.Println("SMTP connection error")
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

## Best Practices

1. **Use App Passwords**: For Gmail, use app-specific passwords, not your main password
2. **Environment Variables**: Store SMTP credentials in environment variables
3. **Queue Long Operations**: Use SendAsync() or queue for bulk emails
4. **HTML Sanitization**: Sanitize user input before adding to HTML body
5. **Rate Limiting**: Respect SMTP provider rate limits
6. **Error Logging**: Always log email send failures
7. **Test Mode**: Use LogMailer in development to avoid sending real emails

## Configuration via Environment

```go
config := &mail.SMTPConfig{
    Host:     os.Getenv("MAIL_HOST"),
    Port:     getEnvInt("MAIL_PORT", 587),
    Username: os.Getenv("MAIL_USERNAME"),
    Password: os.Getenv("MAIL_PASSWORD"),
    From: mail.Address{
        Email: os.Getenv("MAIL_FROM_ADDRESS"),
        Name:  os.Getenv("MAIL_FROM_NAME"),
    },
    UseTLS: getEnvBool("MAIL_USE_TLS", true),
}
```

## Testing

```go
func TestEmailSending(t *testing.T) {
    // Use log mailer for testing
    logger := log.New(os.Stdout, "", log.LstdFlags)
    mailer := mail.NewLogMailer(logger)

    message := mail.NewMessage().
        From("test@app.com", "Test").
        To("user@example.com").
        Subject("Test Email").
        Body("Test body")

    err := mailer.Send(message)
    assert.NoError(t, err)
}
```

## Troubleshooting

### Gmail: "Username and Password not accepted"

- Enable "Less secure app access" (not recommended) OR
- Use App Password: Google Account → Security → App Passwords

### Connection Timeout

- Check firewall rules
- Verify SMTP port is open (587, 465, or 25)
- Try different port numbers

### TLS Errors

- For port 587, set `UseTLS: true`
- For port 465, use SSL (not TLS)
- For port 25, usually no encryption
