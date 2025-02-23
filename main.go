package main

import (
	"fmt"
	"log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"google.golang.org/protobuf/reflect/protoreflect"

	"grpc-xray/httpparser"
	"grpc-xray/models"
	"grpc-xray/renderers"
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

func main() {
	handle, err := pcap.OpenLive("lo0", 1600, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// Filter to capture only gRPC traffic on port 50051
	err = handle.SetBPFFilter("tcp port 50051")
	if err != nil {
		log.Fatal(err)
	}

	modelsCh := make(chan models.RenderModel, 1)
	go renderOutput(modelsCh)
	httpParser := httpparser.New(&modelsCh)

	packetSource := gopacket.NewPacketSource(handle, layers.LayerTypeTCP)
	for {
		select {
		case packet := <-packetSource.Packets():
			if packet == nil {
				return
			}

			err = httpParser.Parse(packet)
			if err != nil {
				fmt.Println("Error parsing packet:", err)
			}

		}
	}
}

func renderOutput(models chan models.RenderModel) {
	renderer := renderers.GetApplicationRenderer()
	for {
		select {
		case model := <-models:
			fmt.Println(renderer.Render(model))
		}
	}
}
