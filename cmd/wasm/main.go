//go:build js
// +build js

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strings"
	"syscall/js"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/net"
)

/*
from:

* `Omri Cohen's` [Run Go In The Browser Using WebAssembly](https://dev.bitolog.com/go-in-the-browser-using-webassembly/)
* `Alessandro Segala` [Go, WebAssembly, HTTP requests and Promises](https://withblue.ink/2020/10/03/go-webassembly-http-requests-and-promises.html)
*/

var hc *http.Client

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	fmt.Println("============================================")
	fmt.Println("init wasm ...")
	fmt.Println("============================================")

	serverName := js.Global().Get("serverName").String()
	fmt.Println("connect to server", serverName)
	js.Global().Set("base64", encodeWrapper())
	js.Global().Set("GoHttp", GoHttp())
	js.Global().Set("GoHttp1", GoHttp1())
	js.Global().Set("GoHttpAsync", GoHttpAsync())
	tp, err := net.NewTransport("http://114.115.218.1:8080", serverName)
	if err != nil {
		fmt.Printf("init webrtc wasm error:%v\n", err)
		return
	}

	fmt.Println("init wasm sucess.")
	hc = &http.Client{
		Transport: tp,
		Timeout:   120 * time.Second,
	}
	<-make(chan bool)
}

func encodeWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return wrap("", "Not enough arguments")
		}
		input := args[0].String()
		return wrap(base64.StdEncoding.EncodeToString([]byte(input)), "")
	})
}

func wrap(encoded string, err string) map[string]interface{} {
	return map[string]interface{}{
		"error":   err,
		"encoded": encoded,
	}
}

// http [Method,url,params]
func GoHttp() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		method := args[0].String()
		url := args[1].String()
		var data string
		if len(args) > 2 && !args[2].IsNull() {
			data = args[2].String()
		}
		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logrus.WithField("stacks", string(debug.Stack())).Errorf("recovered %v", r)
					}
				}()
				var body io.Reader
				if data != "" {
					body = strings.NewReader(data)
				}
				req, err := http.NewRequest(method, url, body)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				if hc == nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New("connecting ...")
					reject.Invoke(errorObject)
					return
				}
				res, err := hc.Do(req)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				defer res.Body.Close()

				data, err := ioutil.ReadAll(res.Body)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}

				m := md5.New()
				io.Copy(m, bytes.NewReader(data))
				fmt.Printf("webrtc md5:%x len:%d", m.Sum(nil), len(data))

				arrayConstructor := js.Global().Get("Uint8Array")
				dataJS := arrayConstructor.New(len(data))
				js.CopyBytesToJS(dataJS, data)

				responseConstructor := js.Global().Get("Response")
				response := responseConstructor.New(dataJS)

				resolve.Invoke(response)
			}()
			return nil
		})
		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}

// http [Method,url,params]
func GoHttp1() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		method := args[0].String()
		url := args[1].String()
		var data string
		if len(args) > 2 && !args[2].IsNull() {
			data = args[2].String()
		}
		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logrus.WithField("stacks", string(debug.Stack())).Errorf("recovered %v", r)
					}
				}()
				var body io.Reader
				if data != "" {
					body = strings.NewReader(data)
				}
				req, err := http.NewRequest(method, url, body)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}

				res, err := http.DefaultClient.Do(req)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				defer res.Body.Close()

				data, err := ioutil.ReadAll(res.Body)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}

				m := md5.New()
				io.Copy(m, bytes.NewReader(data))
				fmt.Printf("http md5:%x len:%d", m.Sum(nil), len(data))

				arrayConstructor := js.Global().Get("Uint8Array")
				dataJS := arrayConstructor.New(len(data))
				js.CopyBytesToJS(dataJS, data)

				responseConstructor := js.Global().Get("Response")
				response := responseConstructor.New(dataJS)

				resolve.Invoke(response)
			}()
			return nil
		})
		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}

func GoHttpAsync() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		method := args[0].String()
		url := args[1].String()
		var data string
		if len(args) > 2 && !args[2].IsNull() {
			data = args[2].String()
		}
		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logrus.WithField("stacks", string(debug.Stack())).Errorf("recovered %v", r)
					}
				}()
				var body io.Reader
				if data != "" {
					body = strings.NewReader(data)
				}
				req, err := http.NewRequest(method, url, body)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				if hc == nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New("connecting ...")
					reject.Invoke(errorObject)
					return
				}
				res, err := hc.Do(req)
				if err != nil {
					// Handle errors: reject the Promise if we have an error
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				underlyingSource := map[string]interface{}{
					"start": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						controller := args[0]

						go func() {
							defer func() {
								if r := recover(); r != nil {
									logrus.WithField("stacks", string(debug.Stack())).Errorf("recovered %v", r)
								}
							}()
							defer res.Body.Close()
							for {
								// Read up to 16KB at a time
								buf := make([]byte, 16384)
								n, err := res.Body.Read(buf)
								if err != nil && err != io.EOF {
									errorConstructor := js.Global().Get("Error")
									errorObject := errorConstructor.New(err.Error())
									controller.Call("error", errorObject)
									return
								}
								if n > 0 {

									arrayConstructor := js.Global().Get("Uint8Array")
									dataJS := arrayConstructor.New(n)
									js.CopyBytesToJS(dataJS, buf[0:n])
									controller.Call("enqueue", dataJS)
								}
								if err == io.EOF {
									controller.Call("close")
									return
								}
							}
						}()

						return nil
					}),

					"cancel": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						res.Body.Close()
						return nil
					}),
				}

				readableStreamConstructor := js.Global().Get("ReadableStream")
				readableStream := readableStreamConstructor.New(underlyingSource)
				responseInitObj := map[string]interface{}{
					"status":     http.StatusOK,
					"statusText": http.StatusText(http.StatusOK),
				}
				responseConstructor := js.Global().Get("Response")
				response := responseConstructor.New(readableStream, responseInitObj)
				resolve.Invoke(response)
			}()
			return nil
		})

		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}
