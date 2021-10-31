package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	pb "github.com/birkj/Miniproject2_Jens/proto"
	"google.golang.org/grpc"
)

const (
	address = "localhost:8080"
)

var client pb.ChittyChatClient
var wait *sync.WaitGroup

func init() {
	wait = &sync.WaitGroup{}
}

func connect(user *pb.User) error {
	var streamerror error
	stream, err := client.CreateStream(context.Background(), &pb.Connect{
		User:   user,
		Active: true,
	})
	if err != nil {
		return fmt.Errorf("connection has failed: %v", err)
	}
	log.Println("Connecting to server. Timestamp: ", user.Time)
	wait.Add(1)
	//Recieve
	go func(str pb.ChittyChat_CreateStreamClient) {
		defer wait.Done()
		for {
			msg, err := str.Recv()
			if err != nil {
				streamerror = fmt.Errorf("error reading message: %v", err)
				break
			}
			sendingUserTime := msg.From.Time
			currentClientTime := user.Time

			user.Time = calcUserTimeLamport(sendingUserTime, currentClientTime)

			log.Println("From: ", msg.From.Name, "Message:", msg.Message, "Timestamp:", user.Time)
		}
	}(stream)
	return streamerror
}

func calcUserTimeLamport(incomming int32, cur int32) int32 {

	if cur > incomming {
		return cur + 1
	} else {
		return incomming + 1
	}
}

func main() {
	timestamp := time.Now()
	done := make(chan int)

	name := flag.String("Name", "Anon", "The name of the user")
	flag.Parse()

	id := sha256.Sum256([]byte(timestamp.String() + *name))

	conn, err := grpc.Dial(address, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("could not connect to service : %v", err)
	}

	client = pb.NewChittyChatClient(conn)
	user := &pb.User{
		Id:     hex.EncodeToString(id[:]),
		Name:   *name,
		Time:   1,
		Active: true,
	}

	connect(user)

	wait.Add(1)
	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			user.Time++
			msg := &pb.ChatMessage{
				Id:      user.Id,
				From:    user,
				Message: scanner.Text(),
			}

			log.Println("Sending message... Timestamp:", user.Time)

			if msg.Message == "exit" {
				msg.Message = msg.From.Name + " has left the chat"
				msg.From.Active = false

				_, err := client.BroadcastMessage(context.Background(), msg)
				if err != nil {
					fmt.Printf("error sending message: %v", err)
					break
				}

				conn.Close()
				close(done)
				break
			}

			_, err := client.BroadcastMessage(context.Background(), msg)
			if err != nil {
				fmt.Printf("error sending message: %v", err)
				break
			}

		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
}
