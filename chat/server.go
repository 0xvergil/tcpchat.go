package chat

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const (
	maxLine      = 4096
	maxNameLen   = 32
	outBuffer    = 128
	writeTimeout = 10 * time.Second
	namePrompt   = "[ENTER YOUR NAME]: "
)

var logo = []string{
	"Welcome to TCP-Chat!",
	"         _nnnn_",
	"        dGGGGMMb",
	"       @p~qp~~qMb",
	"       M|@||@) M|",
	"       @,----.JM|",
	"      JS^\\__/  qKL",
	"     dZP        qKRb",
	"    dZP          qKKb",
	"   fZP            SMMb",
	"   HZM            MMMM",
	"   FqM            MMMM",
	" __| \".        |\\dS\"qML",
	" |    `.       | `' \\Zq",
	"_)      \\.___.,|     .'",
	"\\____   )MMMMMP|   .'",
	"     `-'       `--'",
}

type client struct {
	conn  net.Conn
	name  string
	out   chan string
	wdone chan struct{}
}

func (c *client) send(s string) {
	select {
	case c.out <- s:
	default:
		c.conn.Close()
	}
}

func (c *client) writeLoop() {
	defer close(c.wdone)
	failed := false
	for m := range c.out {
		if failed {
			continue
		}
		c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if _, err := io.WriteString(c.conn, m); err != nil {
			failed = true
			c.conn.Close()
		}
	}
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("accept: %v", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	c := &client{conn: conn, out: make(chan string, outBuffer), wdone: make(chan struct{})}
	go c.writeLoop()
	defer func() {
		close(c.out)
		<-c.wdone
		conn.Close()
	}()

	for _, l := range logo {
		c.send(l + "\n")
	}

	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 1024), maxLine)

	for registered := false; !registered; {
		c.send(namePrompt)
		if !sc.Scan() {
			return
		}
		name := strings.TrimSpace(sc.Text())
		if err := validName(name); err != nil {
			c.send(err.Error() + "\n")
			continue
		}
		c.name = name
		registered = true
	}

	// Messages are handled starting from the next commit; discard for now.
	for sc.Scan() {
	}
	if err := sc.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("read %s: %v", c.name, err)
	}
}

func validName(name string) error {
	switch {
	case name == "":
		return errors.New("Name cannot be empty.")
	case len([]rune(name)) > maxNameLen:
		return fmt.Errorf("Name too long (max %d characters).", maxNameLen)
	case strings.IndexFunc(name, func(r rune) bool { return r < 32 || r == 127 }) >= 0:
		return errors.New("Name contains invalid characters.")
	}
	return nil
}