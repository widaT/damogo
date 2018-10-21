package main

import (
	"log"
	"github.com/widaT/damogo/pb"
	"context"
	"google.golang.org/grpc"
	"fmt"
)

const (
	address     = "localhost:50051"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewFacedbClient(conn)


	ret, err := c.AddUser(context.Background(),&pb.UserInfo{Id:"wida",Feature:&pb.Feature{Feature:[]float32{1.0,33.9}}})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	//log.Printf("Greeting: %s", r.Users)
	fmt.Println(ret)


	r, err := c.Search(context.Background(), &pb.SearchRequest{Group:"wida",Feature:[]float32{111}})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Users)
}