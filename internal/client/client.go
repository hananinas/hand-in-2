package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	pb "hospital/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
	}

	return credentials.NewTLS(config), nil
}

func createClient() pb.SecretSharingServiceClient {

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(tlsCredentials))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	return pb.NewSecretSharingServiceClient(conn)
}

func sendShare(client pb.SecretSharingServiceClient, share *pb.Share, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	r, err := client.SendShare(ctx, share)
	if err != nil {
		log.Fatalf("could not send share: %v", err)
	}
	log.Printf("Client - Acknowledgement: %s", r.GetMessage())
}

func sendOutShare(client pb.SecretSharingServiceClient, share *pb.ShareOut, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	r, err := client.SendShareOut(ctx, share)
	if err != nil {
		log.Fatalf("Client - could not send out: %v", err)
	}
	log.Printf("Client - Acknowledgement: %s", r.GetMessage())
}

func GetAddedOut(client pb.SecretSharingServiceClient, participant string, wg *sync.WaitGroup) int64 {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	log.Printf("Client - Sending GetAddedShares request for participant %s", participant)
	response, err := client.GetAddedShares(ctx, &pb.GetAddedSharesRequest{Participant: participant})
	if err != nil {
		log.Fatalf("Client - could not get added out shares for %s: %v", participant, err)
	}

	return response.AddedShares
}

func generateShares(value int64) (int64, int64, int64) {
	part1 := value / 3
	part2 := value / 3
	part3 := value / 3
	return part1, part2, part3
}

func party1(wg *sync.WaitGroup) {
	defer wg.Done()
	client := createClient()

	x1, x2, x3 := generateShares(30)

	var innerWg sync.WaitGroup
	innerWg.Add(2)
	go sendShare(client, &pb.Share{Part: x2, From: "patient1", To: "patient2"}, &innerWg)
	go sendShare(client, &pb.Share{Part: x3, From: "patient1", To: "patient3"}, &innerWg)
	innerWg.Wait()

	// Compute local result
	innerWg.Add(2)
	var addedShare int64
	go func() {
		defer innerWg.Done()
		addedShare = getAddedShares(client, "patient1", &innerWg)
	}()
	innerWg.Wait()

	out1 := x1 + addedShare //

	innerWg.Add(2)
	go sendOutShare(client, &pb.ShareOut{Data: out1, From: "patient1", To: "patient2"}, &innerWg)
	go sendOutShare(client, &pb.ShareOut{Data: out1, From: "patient1", To: "patient3"}, &innerWg)
	innerWg.Wait()

	innerWg.Add(2)
	var addedOut int64
	go func() {
		defer innerWg.Done()
		addedOut = GetAddedOut(client, "patient1", &innerWg)
	}()
	innerWg.Wait()

	out := out1 + addedOut //

	// Simulate receiving out2 and out3 to reconstruct the final output
	//	finalOutput := reconstructOutput(out1, 0, 0) // Placeholder for received out2 and out3
	log.Printf("Client - Patient 1 final output: %d", out)
}

func party2(wg *sync.WaitGroup) {
	defer wg.Done()
	client := createClient()

	y1, y2, y3 := generateShares(300)

	var innerWg sync.WaitGroup
	innerWg.Add(2)
	go sendShare(client, &pb.Share{Part: y1, From: "patient2", To: "patient1"}, &innerWg)
	go sendShare(client, &pb.Share{Part: y3, From: "patient2", To: "patient3"}, &innerWg)
	innerWg.Wait()

	// Compute local result
	innerWg.Add(2)
	var addedShare int64
	go func() {
		defer innerWg.Done()
		addedShare = getAddedShares(client, "patient2", &innerWg)
	}()
	innerWg.Wait()

	out2 := y2 + addedShare //

	innerWg.Add(2)
	go sendOutShare(client, &pb.ShareOut{Data: out2, From: "patient2", To: "patient1"}, &innerWg)
	go sendOutShare(client, &pb.ShareOut{Data: out2, From: "patient2", To: "patient3"}, &innerWg)
	innerWg.Wait()

	innerWg.Add(2)
	var addedOut int64
	go func() {
		defer innerWg.Done()
		addedOut = GetAddedOut(client, "patient2", &innerWg)
	}()
	innerWg.Wait()

	out := out2 + addedOut //

	// Simulate receiving out2 and out3 to reconstruct the final output
	//	finalOutput := reconstructOutput(out1, 0, 0) // Placeholder for received out2 and out3
	log.Printf("Client - Patient 2 final output: %d", out)
}

func party3(wg *sync.WaitGroup) {
	defer wg.Done()
	client := createClient()

	z1, z2, z3 := generateShares(100)

	var innerWg sync.WaitGroup
	innerWg.Add(2)
	go sendShare(client, &pb.Share{Part: z1, From: "patient3", To: "patient1"}, &innerWg)
	go sendShare(client, &pb.Share{Part: z3, From: "patient3", To: "patient2"}, &innerWg)
	innerWg.Wait()

	// Compute local result
	innerWg.Add(2)
	var addedShare int64
	go func() {
		defer innerWg.Done()
		addedShare = getAddedShares(client, "patient3", &innerWg)
	}()
	innerWg.Wait()

	out3 := z2 + addedShare //

	innerWg.Add(2)
	go sendOutShare(client, &pb.ShareOut{Data: out3, From: "patient3", To: "patient1"}, &innerWg)
	go sendOutShare(client, &pb.ShareOut{Data: out3, From: "patient3", To: "patient2"}, &innerWg)
	innerWg.Wait()

	innerWg.Add(2)
	var addedOut int64
	go func() {
		defer innerWg.Done()
		addedOut = GetAddedOut(client, "patient3", &innerWg)
	}()
	innerWg.Wait()

	// Simulate receiving out2 and out3 to reconstruct the final output
	out := out3 + addedOut

	log.Printf("Client - Patient 3 final output: %d", out)
}

func StartClient() {
	var wg sync.WaitGroup
	wg.Add(3)

	// Start each party as a separate goroutine
	go party1(&wg)
	go party2(&wg)
	go party3(&wg)

	// Wait for all parties to complete
	wg.Wait()
}
