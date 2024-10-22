package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	pb "hospital/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	connPool = sync.Pool{
		New: func() interface{} {
			conn, err := createTLSConnection()
			if err != nil {
				log.Fatalf("Failed to create connection: %v", err)
			}
			return conn
		},
	}
)

func createTLSConnection() (*grpc.ClientConn, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := os.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	creds := credentials.NewTLS(&tls.Config{
		RootCAs: certPool,
	})

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func getClientConn() *grpc.ClientConn {
	return connPool.Get().(*grpc.ClientConn)
}

func releaseClientConn(conn *grpc.ClientConn) {
	connPool.Put(conn)
}

func sendShare(client pb.SecretSharingServiceClient, share *pb.Share, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	r, err := client.SendShare(ctx, share)
	if err != nil {
		log.Fatalf("could not send share: %v", err)
	}
	log.Printf("Client - Acknowledgement: %s", r.GetMessage())
}

func sendOutShare(client pb.SecretSharingServiceClient, share *pb.ShareOut, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	r, err := client.SendShareOut(ctx, share)
	if err != nil {
		log.Fatalf("Client - could not send out: %v", err)
	}
	log.Printf("Client - Acknowledgement: %s", r.GetMessage())
}

func GetAddedOut(client pb.SecretSharingServiceClient, participant string, wg *sync.WaitGroup) int64 {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	log.Printf("Client - Sending GetAddedOut request for participant %s", participant)
	response, err := client.GetAddedOut(ctx, &pb.GetAddedOutRequest{Participant: participant})
	if err != nil {
		log.Fatalf("Client - could not get added shares for %s: %v", participant, err)
	}

	return response.AddedOut
}

func getAddedShares(client pb.SecretSharingServiceClient, participant string, wg *sync.WaitGroup) int64 {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	log.Printf("Client - Sending GetAddedShares request for participant %s", participant)
	response, err := client.GetAddedShares(ctx, &pb.GetAddedSharesRequest{Participant: participant})
	if err != nil {
		log.Fatalf("Client - could not get added out shares for %s: %v", participant, err)
	}

	return response.AddedShares
}

func generateShares(value int64) (int64, int64, int64) {
	// Example share generation logic
	share1 := value / 3
	share2 := value / 3
	share3 := value - share1 - share2
	log.Printf("Generated shares: %d, %d, %d", share1, share2, share3)
	return share1, share2, share3
}

func party1(wg *sync.WaitGroup) {
	defer wg.Done()

	conn := getClientConn()
	defer releaseClientConn(conn)
	client := pb.NewSecretSharingServiceClient(conn)

	x1, x2, x3 := generateShares(30)

	var innerWg sync.WaitGroup
	innerWg.Add(2)
	go sendShare(client, &pb.Share{Part: x2, From: "Alice", To: "Bob"}, &innerWg)
	go sendShare(client, &pb.Share{Part: x3, From: "Alice", To: "Charlie"}, &innerWg)
	innerWg.Wait()

	// Compute local result
	innerWg.Add(2)
	var addedShare int64
	go func() {
		defer innerWg.Done()
		addedShare = getAddedShares(client, "Alice", &innerWg)
	}()
	innerWg.Wait()

	out1 := x1 + addedShare //

	innerWg.Add(2)
	go sendOutShare(client, &pb.ShareOut{Data: out1, From: "Alice", To: "Bob"}, &innerWg)
	go sendOutShare(client, &pb.ShareOut{Data: out1, From: "Alice", To: "Charlie"}, &innerWg)
	innerWg.Wait()

	innerWg.Add(2)
	var addedOut int64
	go func() {
		defer innerWg.Done()
		addedOut = GetAddedOut(client, "Alice", &innerWg)
	}()
	innerWg.Wait()

	out := out1 + addedOut //

	// Simulate receiving out2 and out3 to reconstruct the final output
	//	finalOutput := reconstructOutput(out1, 0, 0) // Placeholder for received out2 and out3
	log.Printf("Client - Patient 1 final output: %d", out)
}

func party2(wg *sync.WaitGroup) {
	defer wg.Done()
	conn := getClientConn()
	defer releaseClientConn(conn)
	client := pb.NewSecretSharingServiceClient(conn)

	y1, y2, y3 := generateShares(300)

	var innerWg sync.WaitGroup
	innerWg.Add(2)
	go sendShare(client, &pb.Share{Part: y1, From: "Bob", To: "Alice"}, &innerWg)
	go sendShare(client, &pb.Share{Part: y3, From: "Bob", To: "Charlie"}, &innerWg)
	innerWg.Wait()

	// Compute local result
	innerWg.Add(2)
	var addedShare int64
	go func() {
		defer innerWg.Done()
		addedShare = getAddedShares(client, "Bob", &innerWg)
	}()
	innerWg.Wait()

	out2 := y2 + addedShare //

	innerWg.Add(2)
	go sendOutShare(client, &pb.ShareOut{Data: out2, From: "Bob", To: "Alice"}, &innerWg)
	go sendOutShare(client, &pb.ShareOut{Data: out2, From: "patient2", To: "Charlie"}, &innerWg)
	innerWg.Wait()

	innerWg.Add(2)
	var addedOut int64
	go func() {
		defer innerWg.Done()
		addedOut = GetAddedOut(client, "Bob", &innerWg)
	}()
	innerWg.Wait()

	out := out2 + addedOut //

	// Simulate receiving out2 and out3 to reconstruct the final output
	//	finalOutput := reconstructOutput(out1, 0, 0) // Placeholder for received out2 and out3
	log.Printf("Client - Patient 2 final output: %d", out)
}

func party3(wg *sync.WaitGroup) {
	defer wg.Done()
	conn := getClientConn()
	defer releaseClientConn(conn)
	client := pb.NewSecretSharingServiceClient(conn)

	z1, z2, z3 := generateShares(30)

	var innerWg sync.WaitGroup
	innerWg.Add(2)
	go sendShare(client, &pb.Share{Part: z1, From: "Charlie", To: "Alice"}, &innerWg)
	go sendShare(client, &pb.Share{Part: z3, From: "Charlie", To: "Bob"}, &innerWg)
	innerWg.Wait()

	// Compute local result
	innerWg.Add(2)
	var addedShare int64
	go func() {
		defer innerWg.Done()
		addedShare = getAddedShares(client, "Charlie", &innerWg)
	}()
	innerWg.Wait()

	out3 := z2 + addedShare //

	innerWg.Add(2)
	go sendOutShare(client, &pb.ShareOut{Data: out3, From: "Charlie", To: "Alice"}, &innerWg)
	go sendOutShare(client, &pb.ShareOut{Data: out3, From: "Charlie", To: "Bob"}, &innerWg)
	innerWg.Wait()

	innerWg.Add(2)
	var addedOut int64
	go func() {
		defer innerWg.Done()
		addedOut = GetAddedOut(client, "Charlie", &innerWg)
	}()
	innerWg.Wait()

	// Simulate receiving out2 and out3 to reconstruct the final output
	out := out3 + addedOut

	log.Printf("Client - Patient 3 final output: %d", out)
}

func StartClient(wg *sync.WaitGroup) {
	defer wg.Done()

	var clientWg sync.WaitGroup
	clientWg.Add(3)

	// Start each party as a separate goroutine
	go party1(&clientWg)
	go party2(&clientWg)
	go party3(&clientWg)

	// Wait for all parties to complete
	clientWg.Wait()
	log.Println("Client has finished.")
}
