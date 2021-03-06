package main

import (
	"context"
	"log"
	"time"

	capi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"

	"github.com/rfyiamcool/grpclb-scheduler"
	"github.com/rfyiamcool/grpclb-scheduler/examples/proto"
	"github.com/rfyiamcool/grpclb-scheduler/registry/consul"
)

var (
	ServiceName = "test"
)

func main() {
	config := &capi.Config{
		Address: "http://127.0.0.1:8500",
	}

	resolver, err := consul.NewResolver(ServiceName, config)
	if err != nil {
		panic(err.Error())
	}

	lb := grpclb.NewBalancer(resolver, grpclb.NewRoundRobinSelector())
	conn, err := grpc.Dial("", grpc.WithInsecure(), grpc.WithBalancer(lb))
	if err != nil {
		log.Printf("grpc dial: %s", err)
		return
	}
	defer conn.Close()

	var (
		client = proto.NewTestClient(conn)
		count  = 10
	)
	for index := 0; index < count; index++ {
		resp, err := client.Say(context.Background(), &proto.SayReq{Content: "consul"})
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf(resp.Content)

		// for debug
		log.Println("active sleep 2s, u can stop a node")
		time.Sleep(2 * time.Second)
	}
}
