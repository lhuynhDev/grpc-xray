package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "example-server/test"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedUserServiceServer
}

// CreateUser handles incoming gRPC requests
func (s *server) CreateUser(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	fmt.Println("Received request:", req)

	// Respond with a simple message
	return &pb.UserResponse{
		Message: fmt.Sprintf("User %s created successfully!", req.Name),
	}, nil
}

func main() {
	// Listen on port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &server{})

	// Enable reflection
	reflection.Register(grpcServer)

	log.Println("gRPC server with reflection listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
