package main

import (
	"net/rpc"
	"log"
	"net"
	"time"
	"sync"
	"fmt"
)

type TestService struct {

}

func (t *TestService) Echo(arg string, result *string) error {
	*result = arg
	return nil
}

func RegisterAndServerTcp()  {
	err := rpc.Register(&TestService{})
	if err != nil {
		log.Fatal(err)
		return
	}
	addr, _ := net.ResolveTCPAddr("tcp", ":1234")
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
			return
		} else {
			rpc.ServeCodec(NewServerCodec(conn))
		}
	}
}

func Test(arg string) (result string, err error) {
	conn, err := net.Dial("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	client := rpc.NewClientWithCodec(NewClientCodec(conn))
	defer client.Close()
	err = client.Call("TestService.Echo", arg, &result)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return result, nil
}

func main() {
	go RegisterAndServerTcp()
	time.Sleep(time.Second)
	wg := new(sync.WaitGroup)
	n := 10
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			srg := fmt.Sprintf("echo i = %v", i)
			result, err := Test(srg)
			if err != nil {
				log.Fatal(err)
			} else {
				fmt.Println(result)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}
