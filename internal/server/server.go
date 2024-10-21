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
	receivedShares map[string]int64 // key is the port and the value is the part
	outShares      map[string]int64
	mu             sync.Mutex
}

// SendShare receives a Share message
func (s *server) SendShare(ctx context.Context, share *pb.Share) (*pb.Ack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// same as y1+z1 and so on
	s.receivedShares[share.To] += share.Part

	return &pb.Ack{Message: "Share received"}, nil
}

func (s *server) SendShareOut(ctx context.Context, share *pb.ShareOut) (*pb.Ack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// same as out2+out3 and so on
	s.outShares[share.To] += share.Data

	return &pb.Ack{Message: "Out received"}, nil
}

func (s *server) GetAddedOut(ctx context.Context, req *pb.GetAddedOutRequest) (*pb.GetAddedOutResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Filter the added shares for the specified patient
	var totalAddedOut int64
	for from, share := range s.outShares {
		if from == req.Participant {
			totalAddedOut = share
		}
	}

	return &pb.GetAddedOutResponse{AddedOut: totalAddedOut}, nil

}

func (s *server) GetAddedShares(ctx context.Context, req *pb.GetAddedSharesRequest) (*pb.GetAddedSharesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("GetAddedShares request for participant %s", req.Participant)

	// Filter the added shares for the specified patient
	var totalAddedShares int64
	for from, share := range s.receivedShares {
		if from == req.Participant {
			totalAddedShares = share
		}
	}

	return &pb.GetAddedSharesResponse{AddedShares: totalAddedShares}, nil
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")
	if err != nil {
		log.Printf("Error loading server certificate and key: %v", err)
		return nil, err
	}

	// Create the credentials and return it
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
	log.Printf("starting server")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	pb.RegisterSecretSharingServiceServer(grpcServer, s)

	log.Printf("server started")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
