package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "github.com/birkj/Miniproject2_Jens/proto"
	"google.golang.org/grpc"
)

const port = ":8080"

var mutex sync.Mutex

type Connection struct { // Creating a struct to contain each connection from clients
	stream pb.ChittyChat_CreateStreamServer // Each connection contains a stream object which is used to send the message during the broadcast function
	id     string
	name   string
	active bool // A bool used to check if a connection is still active
	error  chan error
}

type Server struct {
	pb.UnimplementedChittyChatServer
	Connection  []*Connection
	logicaltime int32
}

func (s *Server) CreateStream(pconn *pb.Connect, stream pb.ChittyChat_CreateStreamServer) error { // Once a user wants to connect, we create a stream object and add their usercredentials to it in a connection stuct
	conn := &Connection{
		id:     pconn.User.Id,
		stream: stream,
		active: true,
		error:  make(chan error),
	}

	//s.logicaltime = calcServerTimeLamport(pconn.User.Time, s.logicaltime)
	s.Connection = append(s.Connection, conn) // connection is added to the server

	_, err := s.BroadcastMessage(context.Background(), &pb.ChatMessage{
		Message: pconn.User.Name + " has connected.",
		From:    pconn.User,
	})
	//log.Println("From:", msg.From.Name, "Message:", msg.Message, "Timestamp:", s.logicaltime)
	if err != nil {
		log.Printf("could send connection message: %v", err)
	}

	return <-conn.error // channel any errors out
}

func (s *Server) BroadcastMessage(ctx context.Context, msg *pb.ChatMessage) (*pb.Empty, error) {
	mutex.Lock()
	defer mutex.Unlock()
	cur := s.logicaltime
	incomming := msg.From.Time
	s.logicaltime = calcServerTimeLamport(cur, incomming)
	log.Println("From:", msg.From.Name, "Message:", msg.Message, "Timestamp:", s.logicaltime)
	s.logicaltime++
	if !msg.From.Active {
		removeConn(msg.From.Id, s)
	}

	log.Println("Broadcasting... to all users. Timestamp:", s.logicaltime)
	for _, conn := range s.Connection {

		go func(msg *pb.ChatMessage, conn *Connection) {

			if conn.active {
				msg.From.Time = s.logicaltime
				//fmt.Print(msg)
				err := conn.stream.Send(msg)

				if err != nil {
					fmt.Println(err)
				}
			}
		}(msg, conn)
	}

	return &pb.Empty{}, nil
}

func removeConn(id string, s *Server) {
	for i, conn := range s.Connection {
		if id == conn.id {
			s.Connection[i].active = false
		}
	}
}

// func (s *Server) CalculateTime(msg *pb.ChatMessage) []int32 {
// 	var times []int32

// 	times = append(times, s.logicaltime)

// 	for _, conn := range s.Connection {
// 		if msg.GetId() == conn.id {
// 			conn.logicaltime += 1
// 		}
// 		times = append(times, conn.logicaltime)
// 	}

// 	return times
// }

func calcServerTimeLamport(cur int32, incomming int32) int32 {

	if cur > incomming {
		return cur + 1
	} else {
		return incomming + 1
	}
}

func main() {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("Error, couldn't create the server %v", err)
	}

	server := Server{logicaltime: 1}
	log.Println("Starting server at port", port, "Timestamp:", server.logicaltime)
	pb.RegisterChittyChatServer(grpcServer, &server)
	grpcServer.Serve(listener)

}
