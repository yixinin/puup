package backend_test

import (
	"fmt"
	"io"
	"net"
	"testing"
)

func TestTcp(t *testing.T) {
	lis, err := net.Listen("tcp", ":8881")
	if err != nil {
		t.Error(err)
		return
	}

	for {
		conn, err := lis.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		go func(conn io.ReadCloser) {
			defer conn.Close()
			var buf = make([]byte, 1500)
			for {
				n, err := conn.Read(buf)
				if err != nil {
					t.Error(err)
					return
				}

				fmt.Println(buf[:n])
			}
		}(conn)
	}
}
