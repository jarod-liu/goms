package rtmp

import (
	"testing"
)

func TestHandshake(t *testing.T) {
	l, err := Listen(":19350")
	if err != nil {
		t.Error(err)
	}
	defer l.Close()

	go func() {
		c, err := Dial("127.0.0.1:19350")
		if err != nil {
			t.Error(err)
		}
		err = c.Handshake()
		if err != nil {
			t.Error(err)
		}
	}()

	c, err := l.Accept()
	if err != nil {
		t.Error(err)
	}
	err = c.Handshake()
	if err != nil {
		t.Error(err)
	}
}
