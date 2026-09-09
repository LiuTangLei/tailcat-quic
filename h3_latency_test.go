package tailcat

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"tailscale.com/tstest/integration"
)

func TestH3SmallRoundTrips(t *testing.T) {
	dm:=integration.RunDERPAndSTUN(t,func(string,...any){},"127.0.0.1")
	s:=&Server{Region:dm.Regions[1],Logf:func(string,...any){}}
	s.OnTCP=func(uint16)func(net.Conn){return func(c net.Conn){defer c.Close();_,_=io.Copy(c,c)}}
	if err:=s.Start();err!=nil{t.Fatal(err)}
	defer s.Close()
	client:=&Client{Server:s.TailcatAddr(),Logf:func(string,...any){}}
	defer client.Close()
	ctx,cancel:=context.WithTimeout(t.Context(),30*time.Second)
	defer cancel()
	c,err:=client.DialTCPPort(ctx,8080)
	if err!=nil{t.Fatal(err)}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(25*time.Second))
	buf:=make([]byte,8)
	started:=time.Now()
	for i:=0;i<60;i++ {
		before:=time.Now()
		if _,err:=c.Write(buf);err!=nil{t.Fatal(err)}
		if _,err:=io.ReadFull(c,buf);err!=nil{t.Fatal(err)}
		if i%15==0 {t.Logf("round=%d elapsed=%v RTT=%v stats=%+v",i,time.Since(started),time.Since(before),client.lb.h3Backend.Snapshot()["connection_stats"])}
	}
	t.Logf("60 round trips: %v; final stats=%+v",time.Since(started),client.lb.h3Backend.Snapshot()["connection_stats"])
	ping,err:=client.DiscoPing(ctx)
	t.Logf("discovery path=%+v error=%v",ping,err)
}
