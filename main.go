package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

var (
	ShutdownSignal = syscall.SIGINT
	RestartSignal  = syscall.SIGHUP
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World!")
}

func waitSignal(l net.Listener) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, ShutdownSignal)
	go func() {
		sig := <-c
		switch sig {
		case ShutdownSignal:
			signal.Stop(c)
			l.Close()
		}
	}()
}

func isMaster() bool {
	return os.Getenv("FD_KEY") == ""
}

func listenTCP(addr string) (*net.TCPListener, error) {
	laddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	l, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func supervise(l *net.TCPListener) error {
	p, err := forkExec(l)
	if err != nil {
		return err
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, RestartSignal)
	for {
		switch sig := <-c; sig {
		case RestartSignal:
			child, err := forkExec(l)
			if err != nil {
				return err
			}
			p.Signal(ShutdownSignal)
			p.Wait()
			p = child
		}
	}
}

func forkExec(l *net.TCPListener) (*os.Process, error) {
	progName, err := exec.LookPath(os.Args[0])
	if err != nil {
		return nil, err
	}
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	f, err := l.File()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	files := []*os.File{os.Stdin, os.Stdout, os.Stderr, f}
	fdEnv := fmt.Sprintf("%s=%d", "FD_KEY", len(files)-1)
	return os.StartProcess(progName, os.Args, &os.ProcAttr{
		Dir:   pwd,
		Env:   append(os.Environ(), fdEnv),
		Files: files,
	})
}

func getFD() (uintptr, error) {
	fdStr := os.Getenv("FD_KEY")
	fd, err := strconv.Atoi(fdStr)
	if err != nil {
		return 0, err
	}
	return uintptr(fd), nil
}

func main() {
	http.HandleFunc("/", indexHandler)
	s := &http.Server{
		Addr:    ":8888",
		Handler: http.DefaultServeMux,
	}

	var wg sync.WaitGroup
	s.ConnState = func(conn net.Conn, state http.ConnState) {
		switch state {
		case http.StateActive:
			wg.Add(1)
		case http.StateIdle:
			wg.Done()
		}
	}

	if isMaster() {
		log.Printf("master pid: %d\n", os.Getpid())
		l, err := listenTCP("localhost:8888")
		if err != nil {
			log.Fatal(err)
		}
		if err = supervise(l); err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("worker pid: %d\n", os.Getpid())
	fd, err := getFD()
	if err != nil {
		log.Fatal(err)
	}
	file := os.NewFile(fd, "listen socket")
	defer file.Close()
	l, err := net.FileListener(file)
	if err != nil {
		log.Fatal(err)
	}
	waitSignal(l)
	if err := s.Serve(l); err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}

