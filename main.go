package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// parseMessage recursively converts a protobuf message to JSON-like structure
func parseMessage(msg protoreflect.Message) map[string]interface{} {
	result := make(map[string]interface{})
	fields := msg.Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		value := msg.Get(field)

		// Handle different field types
		switch field.Kind() {
		case protoreflect.MessageKind: // Nested message
			if field.IsList() {
				// Handle repeated messages
				var list []interface{}
				for j := 0; j < value.List().Len(); j++ {
					list = append(list, parseMessage(value.List().Get(j).Message()))
				}
				result[string(field.Name())] = list
			} else {
				// Handle single nested message
				result[string(field.Name())] = parseMessage(value.Message())
			}
		case protoreflect.StringKind:
			result[string(field.Name())] = value.String()
		case protoreflect.Int32Kind, protoreflect.Int64Kind:
			result[string(field.Name())] = value.Int()
		case protoreflect.BoolKind:
			result[string(field.Name())] = value.Bool()
		case protoreflect.BytesKind:
			result[string(field.Name())] = value.Bytes()
		default:
			result[string(field.Name())] = value.Interface()
		}
	}
	return result
}

// decodeRequest decodes a gRPC request payload dynamically
func decodeRequest(msgDesc protoreflect.MessageDescriptor, data []byte) (map[string]interface{}, error) {
	messageType, err := protoregistry.GlobalTypes.FindMessageByName(msgDesc.FullName())
	if err != nil {
		return nil, fmt.Errorf("could not find message type: %w", err)
	}

	// Create a new instance of the message
	msg := messageType.New().Interface()

	// Unmarshal binary data into the message
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	// Convert message to JSON-like structure
	return parseMessage(msg.ProtoReflect()), nil
}

// getServiceDescriptor retrieves the file descriptor from the reflection API
func getServiceDescriptor(client grpc_reflection_v1alpha.ServerReflectionClient, serviceName string) (*grpc_reflection_v1alpha.FileDescriptorResponse, error) {
	stream, err := client.ServerReflectionInfo(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %w", err)
	}

	req := &grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: serviceName,
		},
	}

	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("failed to send reflection request: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	// Debugging: Print reflection response type
	log.Printf("Received reflection response: %+v", resp)

	fileDescResp, ok := resp.MessageResponse.(*grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %+v", resp.MessageResponse)
	}

	return fileDescResp.FileDescriptorResponse, nil
}

func main() {
	// Connect to the gRPC server
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := grpc_reflection_v1alpha.NewServerReflectionClient(conn)

	// Get the descriptor of the service
	serviceName := "test.UserService"
	fd, err := getServiceDescriptor(client, serviceName)
	if err != nil {
		log.Fatalf("failed to get file descriptor: %v", err)
	}

	// Extract message descriptor (assuming 1st service and 1st method)
	if len(fd.FileDescriptorProto) == 0 {
		log.Fatalf("No file descriptors found for service: %s", serviceName)
	}

	// Normally, you'd need to parse the descriptor into a usable structure.
	// Here, we assume the first method and manually assign a dummy request.
	// In a real-world case, you'd dynamically extract and parse method info.

	// Dummy request payload (replace
	// this with real binary gRPC request data)
	rawData := []byte{} // You need actual binary data from a gRPC request

	msgType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName("test.UserRequest"))
	if err != nil {
		log.Fatalf("failed to find message type: %v", err)
	}

	// Decode the request dynamically
	parsedData, err := decodeRequest(msgType.Descriptor(), rawData)
	if err != nil {
		log.Fatalf("failed to parse request: %v", err)
	}

	// Print the parsed request as JSON
	jsonData, _ := json.MarshalIndent(parsedData, "", "  ")
	fmt.Println(string(jsonData))
}
