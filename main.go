package main

func main() {
	s := NewServer()
	s.ListenAndServe(":1935")
}
