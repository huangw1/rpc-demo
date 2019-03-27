package main

import (
	"github.com/huangw1/rpc-demo/step-3/server"
	"time"
	"github.com/huangw1/rpc-demo/step-3/client"
	"math/rand"
	"context"
	"log"
	"fmt"
)

type Test struct {
}

type Arg struct {
	A int
	B int
}

type Reply struct {
	C int
}

func (t Test) Add(ctx context.Context,arg Arg, reply *Reply) error {
	reply.C = arg.A + arg.B
	return nil
}

func main() {
	s := server.NewSimpleServer(server.DefaultOption)
	err := s.Register(Test{}, make(map[string]string))
	if err != nil {
		panic(err)
	}
	go func() {
		err := s.Serve("tcp", ":7000")
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Second * 2)

	c, err := client.NewSimpleClient("tcp", ":7000", client.DefaultOption)
	if err != nil {
		panic(err)
	}
	arg := Arg{A: rand.Intn(200), B: rand.Intn(100)}
	reply := &Reply{}
	err = c.Call(context.TODO(), "Test.Add", arg, &reply)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Test.Add with param %v equal %v", arg, reply)
	}
}
