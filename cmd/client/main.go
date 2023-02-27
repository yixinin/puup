package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/yixinin/puup/net"
	"github.com/yixinin/puup/stderr"
)

func main() {

	Check()
}

func Download() {
	req, err := http.NewRequest("GET", "http://localhost/share/opi5.png", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	tp, err := net.NewTransport("http://114.115.218.1:8080", "open")
	if err != nil {
		fmt.Println(stderr.Wrap(err))
		return
	}
	hc := http.Client{
		Transport: tp,
	}

	resp, err := hc.Do(req)
	if err != nil {
		fmt.Println(stderr.Wrap(err))
		return
	}
	defer resp.Body.Close()
}

var file1 = `C:\Users\eason\Pictures\opi5.png`
var file2 = `C:\Users\eason\Pictures\http1.png`

func Check() {
	file1 = "swi.txt"
	f1, err := os.Open(file1)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f1.Close()
	file2 = "swr.txt"
	f2, err := os.Open(file2)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f2.Close()
	data1, err := io.ReadAll(f1)
	if err != nil {
		fmt.Println(err)
		return
	}
	data2, err := io.ReadAll(f2)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(len(data1), len(data2))
	for i := range data1 {
		if data1[i] != data2[i] {
			fmt.Println(i)

			fmt.Println(data1[i:i+100], data2[i:i+100])
			return
		}
	}
}
