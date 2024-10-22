package server

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"sync"

	pb "hospital/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// server is used to implement secretsharing.SecretSharingServiceServer
type server struct {
	pb.UnimplementedSecretSharingServiceServer
	receivedShares map[string]int64 // key is the participant and the value is the part
	outShares      map[string]int64
	mu             sync.RWMutex // Use RWMutex for more granular locking
}

// SendShare receives a Share message
func (s *server) SendShare(ctx context.Context, share *pb.Share) (*pb.Ack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.receivedShares[share.To]; !exists {
		s.receivedShares[share.To] = 0
	}
	s.receivedShares[share.To] += share.Part
	log.Printf("Updated receivedShares for %s: %d", share.To, s.receivedShares[share.To])

	return &pb.Ack{Message: "Share received"}, nil
}

func (s *server) SendShareOut(ctx context.Context, share *pb.ShareOut) (*pb.Ack, error) {
	s.mu.Lock()

	if _, exists := s.outShares[share.To]; !exists {
		s.outShares[share.To] = 0
	}
	s.outShares[share.To] += share.Data

	s.mu.Unlock()

	log.Printf("Received out share from %s to %s with value %d", share.From, share.To, share.Data)

	return &pb.Ack{Message: "Out received"}, nil
}

func (s *server) GetAddedOut(ctx context.Context, req *pb.GetAddedOutRequest) (*pb.GetAddedOutResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalAddedOut := s.outShares[req.Participant]
	log.Printf("Returning added out for %s: %d", req.Participant, totalAddedOut)

	return &pb.GetAddedOutResponse{AddedOut: totalAddedOut}, nil
}

func (s *server) GetAddedShares(ctx context.Context, req *pb.GetAddedSharesRequest) (*pb.GetAddedSharesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalAddedShares := s.receivedShares[req.Participant]
	log.Printf("Returning added shares for %s: %d", req.Participant, totalAddedShares)

	return &pb.GetAddedSharesResponse{AddedShares: totalAddedShares}, nil
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")
	if err != nil {
		log.Printf("Error loading server certificate and key: %v", err)
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}

	return credentials.NewTLS(config), nil
}

func StartServer() {
	s := &server{
		receivedShares: make(map[string]int64),
		outShares:      make(map[string]int64),
	}

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(tlsCredentials),
	)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("Starting server")

	pb.RegisterSecretSharingServiceServer(grpcServer, s)

	log.Println("Server started")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
