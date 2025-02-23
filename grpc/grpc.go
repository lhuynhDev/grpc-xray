package grpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"grpc-xray/models"

	"github.com/jhump/protoreflect/grpcreflect"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	maxMsgSize   = 1024 * 1024 * 1024
	emptyMessage = ""
)

// Decode to decode proto messages
func Decode(path string, frame *http2.DataFrame, side int, state *models.GrpcState) (interface{}, error) {
	var dataBuf bytes.Buffer
	buf := frame.Data()
	if len(buf) == 0 {
		return nil, nil
	}

	if state.IsPartialRead {
		buf = append(state.Buf, buf...)
	}

	_ = frame.Header().StreamID
	length := int(binary.BigEndian.Uint32(buf[1:5]))

	compress := buf[0]

	if compress == 1 {
		return nil, nil
	}

	dataBuf.Write(buf[5:])

	if length > maxMsgSize || dataBuf.Len() > maxMsgSize {
		dataBuf.Truncate(0)
		return nil, nil
	}

	if length != dataBuf.Len() {
		state.IsPartialRead = true
		state.Buf = buf

		return nil, nil
	}

	state.IsPartialRead = false
	state.Buf = nil

	data := dataBuf.Bytes()

	defer func() {
		dataBuf.Truncate(0)
	}()

	// if method, ok := protoprovider.GetProtoByPath(path); ok {
	// 	switch side {
	// 	case 1:
	// 		msg := *(method.Request)
	// 		if err := msg.Unmarshal(data); err != nil {
	// 			logrus.Errorf("Error unmarshal request: %s", err.Error())
	// 		}
	// 		return msg.String(), nil
	// 	case 2:
	// 		msg := *(method.Response)
	// 		if err := msg.Unmarshal(data); err != nil {
	// 			logrus.Errorf("Error unmarshal response: %s", err.Error())
	// 		}
	// 		return msg.String(), nil
	// 	}
	// }

	cc, err := grpc.Dial(":50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	newCTX := context.Background()
	client := grpcreflect.NewClientAuto(newCTX, cc)

	sd, err := client.ResolveService("test.UserService")
	// fmt.Println(sd)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("from reflection", sd.GetFullyQualifiedName())
	md := sd.FindMethodByName("CreateUser")
	msgInput := md.GetInputType()
	fmt.Println(msgInput.GetName())
	msg := dynamicpb.NewMessage(msgInput.UnwrapMessage())

	if err := proto.Unmarshal(data, msg); err != nil {
		fmt.Println("Failed to unmarshal gRPC message:", err)
	}

	jsonBody, err := protojson.Marshal(msg)
	if err != nil {
		fmt.Print("Failed to marshal message to JSON:", err)
	}

	fmt.Print(jsonBody)

	return emptyMessage, nil
}
