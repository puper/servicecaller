# servicecaller
Call your local service method by string or json.RawMessage, based on net/rpc and net/rpc/jsonrpc

```
type HelloService struct{}

func (p *HelloService) Hello(request string, reply *string) error {
	*reply = "hello:" + request
	return nil
}

func main() {
	s, _ :=servicecaller.New()
	s.RegisterService("hello", new(HelloService))
	reply := json.RawMessage{}
	err := s.Call("hello.Hello", "world", &reply)
	if err != nil {
		log.Println("callby", err)
	}
	log.Println("callWithString: ", string(reply))
	{
		reply := ""
		s.GetService("hello").(*HelloService).Hello("world", &reply)
		log.Println("callDirectly: ", reply)
	}

}
```