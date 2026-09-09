package tailcat

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"tailscale.com/tstest/integration"
)

// The transport dependency runs the complete nettest.Conn contract. This test
// additionally verifies real tailcat/engine shutdown interrupts pending reads.
func TestH3TCPServerShutdownAndCancellation(t *testing.T) {
	dm:=integration.RunDERPAndSTUN(t,func(string,...any){},"127.0.0.1")
	accepted:=make(chan net.Conn,1)
	hold:=make(chan struct{})
	s:=&Server{Region:dm.Regions[1],Logf:func(string,...any){}}
	s.OnTCP=func(port uint16)func(net.Conn){
		if port!=8080{return nil}
		return func(c net.Conn){accepted<-c;<-hold}
	}
	if err:=s.Start();err!=nil{t.Fatal(err)}
	defer s.Close()
	defer close(hold)
	client:=&Client{Server:s.TailcatAddr(),Logf:func(string,...any){}}
	defer client.Close()
	ctx,cancel:=context.WithTimeout(t.Context(),15*time.Second)
	defer cancel()
	c,err:=client.DialTCPPort(ctx,8080)
	if err!=nil{t.Fatal(err)}
	defer c.Close()
	server:=<-accepted
	defer server.Close()
	blocked:=make(chan error,1)
	go func(){buf:=make([]byte,16);_,err:=c.Read(buf);blocked<-err}()
	var wg sync.WaitGroup
	for i:=0;i<4;i++ {wg.Add(1);go func(){defer wg.Done();s.Close()}()}
	wg.Wait()
	select{case err:=<-blocked:if err==nil{t.Fatal("server shutdown did not interrupt read")};case <-ctx.Done():t.Fatal("read stuck after shutdown")}
	canceled,stop:=context.WithCancel(t.Context());stop()
	bad,err:=client.DialTCPPort(canceled,8080)
	if bad!=nil {bad.Close();t.Fatal("canceled dial opened a stream")}
	if err==nil || (!errors.Is(err,context.Canceled) && !errors.Is(err,net.ErrClosed)) {t.Fatalf("unexpected canceled dial error: %v",err)}
}
