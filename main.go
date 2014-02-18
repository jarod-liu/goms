package main

import (
	"github.com/jarod/goms/rtmp"
)

func main() {
	s := rtmp.NewServer()
	s.ListenAndServe(":1935")
}
