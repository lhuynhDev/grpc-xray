package main

import (
	"encoding/json"
	"fmt"
	"log"

	"go.starlark.net/lib/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// parseMessage recursively converts a protobuf message to JSON-like structure
func parseMessage(msg protoreflect.Message) map[string]interface{} {
	result := make(map[string]interface{})
	fields := msg.Descriptor().Fields()

	fields.Range(func(i int, field protoreflect.FieldDescriptor) bool {
		value := msg.Get(field)

		// Handle different field types
		switch field.Kind() {
		case protoreflect.MessageKind: // Nested message
			if field.IsList() {
				// Handle repeated messages
				var list []interface{}
				for i := 0; i < value.List().Len(); i++ {
					list = append(list, parseMessage(value.List().Get(i).Message()))
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

		return true
	})

	return result
}

// decodeRequest takes a binary gRPC request payload and decodes it using reflection
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

func main() {

	// Connect to the server
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := grpc_reflection_v1alpha.NewServerReflectionClient(conn)

	// Get the descriptor of the service
	serviceName := "example.Service"
	fd, err := getServiceDescriptor(client, serviceName)
	if err != nil {
		log.Fatalf("failed to get file descriptor: %v", err)
	}

	// Extract message descriptor (assuming 1st service and 1st method)
	serviceDesc := fd.Service[0]
	methodDesc := serviceDesc.Method[0]
	messageType := methodDesc.InputType

	// Decode the request dynamically
	parsedData, err := decodeRequest(protoregistry.GlobalFiles.FindMessageByName(protoreflect.FullName(messageType)), rawData)
	if err != nil {
		log.Fatalf("failed to parse request: %v", err)
	}

	// Print the parsed request as JSON
	jsonData, _ := json.MarshalIndent(parsedData, "", "  ")
	fmt.Println(string(jsonData))
}
