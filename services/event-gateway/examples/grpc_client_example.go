package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/distributed-event-processor/shared/proto/events/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	serverAddress = "localhost:50051" // Standard gRPC port (changed from 9090 to avoid Prometheus conflict)
)

func main() {
	// Connect to the gRPC server
	conn, err := grpc.NewClient(
		serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create client
	client := pb.NewEventGatewayClient(conn)

	// Test all RPC methods
	testHealthCheck(client)
	testIngestEvent(client)
	testIngestEventBatch(client)
	testValidateEvent(client)
	testStreamEvents(client)
}

func testHealthCheck(client pb.EventGatewayClient) {
	fmt.Println("\n=== Testing HealthCheck ===")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{
		Detailed: true,
	})
	if err != nil {
		log.Fatalf("HealthCheck failed: %v", err)
	}

	fmt.Printf("Status: %v\n", resp.Status)
	fmt.Printf("Version: %s\n", resp.Version)
	fmt.Printf("Components: %v\n", resp.Components)
}

func testIngestEvent(client pb.EventGatewayClient) {
	fmt.Println("\n=== Testing IngestEvent ===")

	// Create event data
	eventData, _ := structpb.NewStruct(map[string]interface{}{
		"user_id": "12345",
		"action":  "login",
		"ip":      "192.168.1.1",
	})

	// Create context with metadata (request ID)
	ctx := context.Background()
	md := metadata.Pairs("x-request-id", "test-request-123")
	ctx = metadata.NewOutgoingContext(ctx, md)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.IngestEvent(ctx, &pb.IngestEventRequest{
		Event: &pb.Event{
			Type:          "user.login",
			Source:        "auth-service",
			TenantId:      "tenant-1",
			Data:          eventData,
			Timestamp:     timestamppb.Now(),
			SchemaVersion: "1.0",
			CorrelationId: "correlation-123",
			Priority:      5,
			Metadata: map[string]string{
				"region": "us-east-1",
			},
		},
		WaitForAck: true,
	})

	if err != nil {
		log.Fatalf("IngestEvent failed: %v", err)
	}

	fmt.Printf("Event ID: %s\n", resp.EventId)
	fmt.Printf("Request ID: %s\n", resp.RequestId)
	fmt.Printf("Partition: %d, Offset: %d\n", resp.Partition, resp.Offset)
	fmt.Printf("Status: %v\n", resp.Status)
}

func testIngestEventBatch(client pb.EventGatewayClient) {
	fmt.Println("\n=== Testing IngestEventBatch ===")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create batch of events
	events := make([]*pb.Event, 3)
	for i := 0; i < 3; i++ {
		eventData, _ := structpb.NewStruct(map[string]interface{}{
			"batch_index": i,
			"message":     fmt.Sprintf("Event %d", i),
		})

		events[i] = &pb.Event{
			Type:      fmt.Sprintf("batch.event.%d", i),
			Source:    "batch-service",
			TenantId:  "tenant-1",
			Data:      eventData,
			Timestamp: timestamppb.Now(),
			Priority:  int32(i),
		}
	}

	resp, err := client.IngestEventBatch(ctx, &pb.IngestEventBatchRequest{
		Events:     events,
		WaitForAck: true,
		FailFast:   false,
	})

	if err != nil {
		log.Fatalf("IngestEventBatch failed: %v", err)
	}

	fmt.Printf("Success count: %d\n", resp.SuccessCount)
	fmt.Printf("Failure count: %d\n", resp.FailureCount)
	fmt.Printf("Processing time: %d ms\n", resp.ProcessingTimeMs)
	fmt.Printf("Request ID: %s\n", resp.RequestId)

	// Print individual results
	for i, result := range resp.Results {
		fmt.Printf("  Event %d: %s (status: %v)\n", i, result.EventId, result.Status)
	}
}

func testValidateEvent(client pb.EventGatewayClient) {
	fmt.Println("\n=== Testing ValidateEvent ===")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventData, _ := structpb.NewStruct(map[string]interface{}{
		"test": "data",
	})

	// Test valid event
	resp, err := client.ValidateEvent(ctx, &pb.ValidateEventRequest{
		Event: &pb.Event{
			Type:   "validation.test",
			Source: "test-client",
			Data:   eventData,
		},
		ValidateSchema: false,
	})

	if err != nil {
		log.Fatalf("ValidateEvent failed: %v", err)
	}

	fmt.Printf("Valid: %v\n", resp.IsValid)
	if len(resp.Errors) > 0 {
		fmt.Println("Errors:")
		for _, e := range resp.Errors {
			fmt.Printf("  - %s: %s (%s)\n", e.Field, e.Message, e.Code)
		}
	}

	// Test invalid event (missing required fields)
	fmt.Println("\nTesting invalid event:")
	resp2, _ := client.ValidateEvent(ctx, &pb.ValidateEventRequest{
		Event: &pb.Event{
			Type: "", // Missing type
		},
	})

	fmt.Printf("Valid: %v\n", resp2.IsValid)
	if len(resp2.Errors) > 0 {
		fmt.Println("Errors:")
		for _, e := range resp2.Errors {
			fmt.Printf("  - %s: %s (%s)\n", e.Field, e.Message, e.Code)
		}
	}
}

func testStreamEvents(client pb.EventGatewayClient) {
	fmt.Println("\n=== Testing StreamEvents (Bidirectional Streaming) ===")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.StreamEvents(ctx)
	if err != nil {
		log.Fatalf("StreamEvents failed: %v", err)
	}

	// Start goroutine to receive responses
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				fmt.Println("Stream closed by server")
				return
			}
			if err != nil {
				log.Printf("Stream receive error: %v", err)
				return
			}

			switch msg := resp.Message.(type) {
			case *pb.StreamEventResponse_Ack:
				fmt.Printf("  ✓ ACK: Event %s (partition: %d, offset: %d)\n",
					msg.Ack.EventId, msg.Ack.Partition, msg.Ack.Offset)
			case *pb.StreamEventResponse_Pong:
				fmt.Printf("  ↔ PONG: %v\n", msg.Pong.Timestamp.AsTime())
			case *pb.StreamEventResponse_Status:
				fmt.Printf("  ! STATUS: %s - %s\n", msg.Status.Code, msg.Status.Message)
			}
		}
	}()

	// Send stream configuration
	if err := stream.Send(&pb.StreamEventRequest{
		Message: &pb.StreamEventRequest_Config{
			Config: &pb.StreamConfig{
				EnableCompression: true,
				BatchSize:         10,
				FlushIntervalMs:   100,
			},
		},
	}); err != nil {
		log.Fatalf("Failed to send config: %v", err)
	}

	// Send some events
	for i := 0; i < 5; i++ {
		eventData, _ := structpb.NewStruct(map[string]interface{}{
			"stream_index": i,
			"timestamp":    time.Now().Unix(),
		})

		err := stream.Send(&pb.StreamEventRequest{
			Message: &pb.StreamEventRequest_Event{
				Event: &pb.Event{
					Type:      "stream.event",
					Source:    "stream-client",
					TenantId:  "tenant-1",
					Data:      eventData,
					Timestamp: timestamppb.Now(),
					Priority:  int32(i),
				},
			},
		})

		if err != nil {
			log.Printf("Failed to send event: %v", err)
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Send ping
	if err := stream.Send(&pb.StreamEventRequest{
		Message: &pb.StreamEventRequest_Ping{
			Ping: &pb.Ping{
				Timestamp: timestamppb.Now(),
			},
		},
	}); err != nil {
		log.Fatalf("Failed to send ping: %v", err)
	}

	// Wait a bit for responses
	time.Sleep(2 * time.Second)

	// Close the stream
	if err := stream.CloseSend(); err != nil {
		log.Printf("Failed to close stream: %v", err)
	}

	fmt.Println("Stream test completed")
}
