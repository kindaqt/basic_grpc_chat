package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "example.com/go-chat-grpc/chat"
	"example.com/go-chat-grpc/chat_server/config"
	"google.golang.org/grpc"
)

func NewChatServer() *ChatServer {
	return &ChatServer{
		Connections: make(map[string]*Connection),
	}
}

type ChatServer struct {
	pb.UnimplementedChatServer
	Connections map[string]*Connection
}

type Connection struct {
	stream pb.Chat_LoginServer
	id     string
	active bool
	err    chan error
}

func (s *ChatServer) Login(pconn *pb.Connection, stream pb.Chat_LoginServer) error {
	log.Printf("Received login request: userName=%s, userID=%s", pconn.User.GetUserName(), pconn.User.GetId())

	conn := &Connection{
		stream: stream,
		id:     pconn.User.Id,
		err:    make(chan error),
	}

	s.Connections[pconn.User.Id] = conn

	return <-conn.err
}

func (s *ChatServer) Logout(ctx context.Context, user *pb.User) (*pb.Close, error) {
	log.Printf("Received logout request: userName=%s, userID=%s", user.GetUserName(), user.GetId())

	userID := user.GetId()
	if _, ok := s.Connections[userID]; !ok {
		return &pb.Close{}, fmt.Errorf("user not found: userID=%s", userID)
	}
	delete(s.Connections, userID)

	return &pb.Close{}, nil
}

func (s *ChatServer) SendMessage(ctx context.Context, msg *pb.Message) (*pb.Close, error) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	for _, conn := range s.Connections {
		log.Println(conn.id)
		wait.Add(1)

		go func(msg *pb.Message, conn *Connection) {
			defer wait.Done()

			log.Printf("Sending message %v to user %v", msg.Id, conn.id)
			if err := conn.stream.Send(msg); err != nil {
				log.Printf("Error with stream %v: %v", conn.id, err)
				conn.active = false
				conn.err <- err
			}
		}(msg, conn)
	}

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
	return &pb.Close{}, nil
}

func (s *ChatServer) Run(port string) error {
	// TODO: use secure connection
	log.Printf("Starting server on port %v", port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Printf("failed to listen on port %s: %v", port, err)
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterChatServer(grpcServer, s)
	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
		return err
	}

	return nil
}

func main() {
	chatServer := NewChatServer()
	if err := chatServer.Run(fmt.Sprintf(":%d", config.ServerPort)); err != nil {
		log.Fatalf("Failed to start chat server: %v", err)
	}
}
