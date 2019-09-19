// +build js,wasm

package sctp

import (
	"fmt"
	"net"
	"syscall/js"
	"time"
)

type CatalystConn struct {
	packetChan chan []byte
	domUDP     js.Value
    domUDPProxy     js.Value
	addr       net.Addr
}

func (c *CatalystConn) Write(p []byte) (int, error) {
    ui8 := make([]uint8, len(p))
    for i, b := range p {
      ui8[i] = b
    }
    jsArray := js.Global().Get("Uint8Array").New(len(ui8))
    js.CopyBytesToJS(jsArray, ui8)
    c.domUDPProxy.Call("send", jsArray)
    return 1, nil
}

func (c *CatalystConn) WriteTo(p []byte, _ net.Addr) (int, error) {
  return c.Write(p)
}

func (c *CatalystConn) Read(p []byte) (n int, err error) {
  recvd := <-c.packetChan
  copied := copy(p, recvd)
  return copied, nil
}

func (c *CatalystConn) ReadFrom(p []byte) (n int, _ net.Addr, err error) {
	recvd := <-c.packetChan
	copied := copy(p, recvd)
	return copied, c.addr, nil
}

func (c *CatalystConn) Close() error {
	c.domUDP.Call("close")
	return nil
}

func (c *CatalystConn) LocalAddr() net.Addr {
	return nil
}

func (c *CatalystConn) RemoteAddr() net.Addr {
  // this is wrong, but note that Catalyst always connects to
  // the serving host and port.
  return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1337}
}

func (c *CatalystConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *CatalystConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *CatalystConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func NewCatalystConn(addr net.Addr) *CatalystConn {
	packetChan := make(chan []byte, 100)
	domUDP := js.Global().Get("document").Get("realUdp")
    domUDPProxy := js.Global().Get("document").Get("udp")

	conn := &CatalystConn{
		packetChan: packetChan,
		domUDP:     domUDP,
        domUDPProxy:     domUDPProxy,
		addr:       addr,
	}
    enqueue := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        int8arrayWrapper := js.Global().Get("Uint8Array").New(args[0].Get("data"))
        value := make([]byte, int8arrayWrapper.Get("byteLength").Int())
        js.CopyBytesToGo(value, int8arrayWrapper);
        //a := js.TypedArrayOf(value)
        //a.Call("set", int8arrayWrapper)
        //a.Release()
        packetChan <-value
        return nil
	})
    var onclose js.Func
    onclose = js.FuncOf(func(this js.Value, args []js.Value) interface{}{
        fmt.Println("CLOSED")
        panic("CLOSED")
        onclose.Release()
        return nil
    })

	domUDP.Set("onmessage", enqueue)
	domUDP.Set("onclose", onclose)
	return conn
}
