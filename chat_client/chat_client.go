package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "example.com/go-chat-grpc/chat"
	"google.golang.org/grpc"
)

const (
	defaultServerAddress = "localhost:50051"
)

func connectToServer(serverAddress string) (*grpc.ClientConn, error) {
	log.Printf("Connecting to server at %s...", serverAddress)

	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}

	log.Printf("Connected to server at %s", serverAddress)

	return conn, nil
}

func parseFlags() (*pb.User, string) {
	name := flag.String("name", "Anonymous", "")
	flag.Parse()
	serverAddress := flag.String("address", defaultServerAddress, "")
	flag.Parse()

	timestamp := time.Now()
	id := sha256.Sum256([]byte(timestamp.String() + *name))
	user := &pb.User{
		Id:       hex.EncodeToString(id[:]),
		UserName: *name,
	}

	log.Printf("New User: name=%s, id=%s", user.GetUserName(), user.GetId())

	return user, *serverAddress
}

func login(user *pb.User, client pb.ChatClient) (pb.Chat_LoginClient, error) {
	log.Printf("Creating stream for user: %v", user)
	stream, err := client.Login(context.Background(), &pb.Connection{
		User:   user,
		Active: true,
	})
	if err != nil {
		log.Fatalf("Failed to connect user: %v", err)
		return nil, err
	}
	log.Println("Created stream")

	return stream, nil
}

func createReciever(stream pb.Chat_LoginClient, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	for {
		msg, err := stream.Recv()

		if err != nil {
			log.Printf("Error while reading message: %v", err)
			return err
		}

		log.Printf("%v : %s", msg.User.UserName, msg.Message)
	}
}

func createProducer(client pb.ChatClient, user *pb.User, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer func() {
		log.Println("Closing Producer")
		wg.Done()
	}()

	scanner := bufio.NewScanner(os.Stdin)
	ts := time.Now()
	msgID := sha256.Sum256([]byte(ts.String() + user.UserName))

	for scanner.Scan() {
		msg := &pb.Message{
			Id:        hex.EncodeToString(msgID[:]),
			User:      user,
			Message:   scanner.Text(),
			Timestamp: ts.String(),
		}

		_, err := client.SendMessage(context.Background(), msg)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			return err
		}
	}

	return nil
}

func logout(client pb.ChatClient, user *pb.User) error {
	log.Printf("Logging %s out...", user.UserName)
	if _, err := client.Logout(context.Background(), user); err != nil {
		log.Printf("Error while cleaning up: %v", err)
		return err
	}
	log.Printf("%s logged out", user.UserName)
	return nil
}

func main() {
	log.Println("Starting client...")

	user, serverAddress := parseFlags()

	conn, err := connectToServer(serverAddress)
	if err != nil {
		log.Fatalf("Could not create client connection")
	}

	client := pb.NewChatClient(conn)
	stream, err := login(user, client)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}

	var wg = sync.WaitGroup{}
	var done = make(chan int)

	// Start Receiver
	go createReciever(stream, &wg)
	// Start Producer
	go createProducer(client, user, &wg)

	log.Println("Start chatting!")

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logout(client, user)
		os.Exit(1)
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	<-done
}
