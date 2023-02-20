package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/backend"
	"github.com/yixinin/puup/browser"
	"github.com/yixinin/puup/frontend"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/server"
)

var (
	runServer  bool
	runBack    bool
	runFront   bool
	runBrowser bool
)

var (
	debugLevel bool
	logfile    string
)

var shareDir string
var cfgFilename string

type funcremove struct {
}

func (funcremove) Levels() []logrus.Level {
	return logrus.AllLevels
}
func (funcremove) Fire(e *logrus.Entry) error {
	if e.Data == nil {
		return nil
	}
	if e.Caller == nil {
		return nil
	}

	e.Data["file"] = fmt.Sprintf("%s:%d", e.Caller.File, e.Caller.Line)
	e.Caller = nil
	return nil
}

func Init() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		// PrettyPrint: true,
	})
	logrus.AddHook(funcremove{})
	logrus.SetReportCaller(true)
	if debugLevel {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}

func main() {
	flag.BoolVar(&runServer, "s", false, "run server")
	flag.BoolVar(&runBack, "b", false, "run back")
	flag.BoolVar(&runFront, "f", false, "run front")
	flag.StringVar(&cfgFilename, "c", "puup.yaml", "config file name")
	flag.BoolVar(&runBrowser, "br", false, "run browser")
	flag.StringVar(&logfile, "log", "", "log to filename")
	flag.BoolVar(&debugLevel, "debug", false, "log debug mode")
	flag.StringVar(&shareDir, "share", ".", "fileserver dir")
	flag.Parse()
	Init()
	if logfile != "" {
		ext := filepath.Ext(logfile)
		var old = fmt.Sprintf("%s_bak%s", logfile[:len(logfile)-len(ext)], ext)
		os.Remove(old)
		os.Rename(logfile, old)

		f, err := os.Create(logfile)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		logrus.SetOutput(f)
	}

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	if runServer {
		wg.Add(1)
		s := server.NewServer()
		go func() {
			defer wg.Done()
			if err := s.Run(ctx); err != nil {
				logrus.Error(err)
			}
		}()
	}
	if runBack {
		wg.Add(1)
		var b, err = backend.NewBackend(cfgFilename)
		if err != nil {
			logrus.Error(err)
			return
		}
		conn.GoFunc(ctx, func(ctx context.Context) error {
			defer wg.Done()
			return b.Run(ctx)
		})
	}
	if runFront {
		wg.Add(1)
		var f, err = frontend.NewFrontEnd(cfgFilename)
		if err != nil {
			logrus.Error(err)
			return
		}
		conn.GoFunc(ctx, func(ctx context.Context) error {
			defer wg.Done()
			return f.Run(ctx)
		})
	}
	if runBrowser {
		wg.Add(1)
		go func() {
			defer wg.Done()
			browser.RunBrowser()
		}()
	}
	var ch = make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	var exit = make(chan struct{})
	go func() {
		wg.Wait()
		close(exit)
	}()

	select {
	case <-ch:
		cancel()
		logrus.Infoln("receive interrupt, wait exit ...")
	case <-exit:
	}

	select {
	case <-time.After(time.Second):
		logrus.Error("wait exit timeout, force quit")
	case <-exit:
		logrus.Infoln("all process done, exit.")
	}
}
