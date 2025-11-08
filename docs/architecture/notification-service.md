# Notification Service Architecture

## Overview

The Notification Service is a Spring Boot microservice responsible for delivering multi-channel notifications triggered by events in the event processing pipeline. It provides a unified interface for sending notifications across various channels while ensuring reliable delivery and tracking.

## 🎯 Responsibilities

- **Multi-channel delivery**: Email, SMS, Slack, webhook notifications
- **Template management**: Dynamic content generation using templates
- **Async processing**: Non-blocking notification processing with queues
- **Delivery tracking**: Status tracking, retry mechanisms, and failure handling
- **Event integration**: Listen to processing pipeline events for trigger-based notifications

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Event Bus     │───▶│  Notification    │───▶│   Channel       │
│   (Kafka)       │    │   Service        │    │   Adapters      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                         │
                       ┌─────────────────┐               ▼
                       │   Template      │    ┌─────────────────┐
                       │   Engine        │    │   External      │
                       └─────────────────┘    │   Services      │
                                │             │                 │
                       ┌─────────────────┐    │ • SMTP Server   │
                       │   Database      │    │ • SMS Gateway   │
                       │ • Templates     │    │ • Slack API     │
                       │ • Delivery Log  │    │ • Webhooks      │
                       │ • User Prefs    │    └─────────────────┘
                       └─────────────────┘
```

## 🔧 Core Components

### 1. Event Listeners

- **Kafka consumers** for processing pipeline events
- **Event filtering** based on notification rules
- **Event transformation** to notification context

### 2. Template Engine

- **Thymeleaf** for HTML email templates
- **FreeMarker** for text templates
- **Variable substitution** from event data
- **Template versioning** and management

### 3. Channel Adapters

- **Email Adapter**: SMTP integration with JavaMail
- **SMS Adapter**: Twilio/AWS SNS integration
- **Slack Adapter**: Slack Web API integration
- **Webhook Adapter**: HTTP POST notifications

### 4. Delivery Management

- **Async processing** with Spring's @Async
- **Retry mechanism** with exponential backoff
- **Dead letter queue** for failed notifications
- **Delivery status tracking**

## 📊 Spring Boot Learning Areas

### Core Spring Boot Features

- **Spring Boot Starter Web**: REST API for notification management
- **Spring Boot Starter Data JPA**: Template and delivery log persistence
- **Spring Boot Starter Mail**: Email notification support
- **Spring Boot Starter Validation**: Input validation
- **Spring Boot Starter Actuator**: Health checks and metrics

### Advanced Features

- **Spring Kafka**: Event-driven notification triggers
- **Spring Async**: Non-blocking notification processing
- **Spring Retry**: Configurable retry policies
- **Spring Scheduling**: Cleanup tasks and batch processing
- **Spring Security**: API authentication and authorization

### Integration Patterns

- **Template Method Pattern**: Channel-specific implementations
- **Strategy Pattern**: Different delivery strategies
- **Observer Pattern**: Event-driven notifications
- **Circuit Breaker**: External service fault tolerance

## 🚀 API Endpoints

### Template Management

```http
POST   /api/v1/templates                 # Create template
GET    /api/v1/templates                 # List templates
GET    /api/v1/templates/{id}            # Get template
PUT    /api/v1/templates/{id}            # Update template
DELETE /api/v1/templates/{id}            # Delete template
```

### Notification Management

```http
POST   /api/v1/notifications             # Send notification
GET    /api/v1/notifications             # List notifications
GET    /api/v1/notifications/{id}        # Get notification status
GET    /api/v1/notifications/{id}/retry  # Retry failed notification
```

### User Preferences

```http
GET    /api/v1/users/{userId}/preferences     # Get user notification preferences
PUT    /api/v1/users/{userId}/preferences     # Update preferences
```

## 🗄️ Data Models

### Notification Template

```java
@Entity
public class NotificationTemplate {
    @Id
    private String id;

    private String name;
    private String subject;
    private String bodyTemplate;
    private NotificationChannel channel;
    private TemplateType type; // EMAIL, SMS, SLACK

    private String version;
    private boolean active;

    @CreationTimestamp
    private LocalDateTime createdAt;

    @UpdateTimestamp
    private LocalDateTime updatedAt;
}
```

### Notification Log

```java
@Entity
public class NotificationLog {
    @Id
    private String id;

    private String templateId;
    private String recipient;
    private NotificationChannel channel;
    private NotificationStatus status;

    private String eventId;
    private String payload;
    private String errorMessage;

    private int retryCount;
    private LocalDateTime scheduledAt;
    private LocalDateTime sentAt;

    @CreationTimestamp
    private LocalDateTime createdAt;
}
```

### User Preferences

```java
@Entity
public class UserNotificationPreference {
    @Id
    private String userId;

    private boolean emailEnabled;
    private boolean smsEnabled;
    private boolean slackEnabled;

    private String emailAddress;
    private String phoneNumber;
    private String slackUserId;

    private Set<EventType> subscribedEvents;

    @UpdateTimestamp
    private LocalDateTime updatedAt;
}
```

## ⚡ Event Integration

### Kafka Event Listeners

```java
@KafkaListener(topics = "processing-events")
public void handleProcessingEvent(ProcessingEvent event) {
    if (shouldNotify(event)) {
        NotificationContext context = createNotificationContext(event);
        notificationService.sendAsync(context);
    }
}

@KafkaListener(topics = "alert-events")
public void handleAlertEvent(AlertEvent event) {
    NotificationContext context = createAlertNotification(event);
    notificationService.sendImmediate(context);
}
```

### Event Types

- **Processing milestones**: Batch completion, error thresholds
- **System alerts**: Service down, high latency, error spikes
- **Data quality**: Schema validation failures, data anomalies
- **Business events**: Custom rule triggers, pattern matches

## 🔄 Delivery Guarantees

### At-Least-Once Delivery

- **Persistent storage** of notification requests
- **Idempotent operations** with unique message IDs
- **Retry mechanisms** with configurable policies
- **Dead letter queues** for manual intervention

### Delivery Status Tracking

- **PENDING**: Queued for delivery
- **PROCESSING**: Currently being sent
- **SENT**: Successfully delivered
- **FAILED**: Delivery failed (will retry)
- **DEAD_LETTER**: Maximum retries exceeded

## 📈 Performance Characteristics

### Throughput Targets

- **1000+ notifications/second** processing capacity
- **Sub-second delivery** for high-priority notifications
- **Horizontal scaling** with multiple instances

### Reliability Features

- **Circuit breakers** for external service failures
- **Bulk operations** for efficiency
- **Rate limiting** to prevent spam
- **Graceful degradation** when services are unavailable

## 🎓 Learning Outcomes

After implementing the Notification Service, you'll have mastered:

### Spring Boot Intermediate Concepts

- Event-driven architecture with Kafka
- Async processing and thread management
- Template engines and content generation
- External service integration patterns
- Error handling and retry mechanisms

### Enterprise Patterns

- Multi-tenant notification preferences
- Audit logging and compliance
- Configuration externalization
- Health checks and monitoring
- Security and authentication

### Integration Skills

- Third-party API integration (Twilio, Slack)
- Email server configuration
- Webhook delivery patterns
- Database design for audit trails
- Message queue patterns

This service provides comprehensive Spring Boot learning while solving real-world notification challenges in an event-driven architecture.

