package chat

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	maxClients   = 10
	maxLine      = 4096
	maxNameLen   = 32
	outBuffer    = 128
	writeTimeout = 10 * time.Second
	timeLayout   = "2006-01-02 15:04:05"
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

type Server struct {
	mu      sync.Mutex
	clients map[*client]struct{} // clients that completed the name step
	active  int                  // every open connection, named or not
	history []string
}

func NewServer() *Server {
	return &Server{clients: make(map[*client]struct{})}
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
		if !s.reserve() {
			conn.SetWriteDeadline(time.Now().Add(time.Second))
			io.WriteString(conn, "Chat is full (max 10 connections). Try again later.\n")
			conn.Close()
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) reserve() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active >= maxClients {
		return false
	}
	s.active++
	return true
}

func (s *Server) release() {
	s.mu.Lock()
	s.active--
	s.mu.Unlock()
}

func (s *Server) handle(conn net.Conn) {
	c := &client{conn: conn, out: make(chan string, outBuffer), wdone: make(chan struct{})}
	go c.writeLoop()
	defer func() {
		s.leave(c)
		close(c.out)
		<-c.wdone
		conn.Close()
		s.release()
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
		if err := s.join(c, strings.TrimSpace(sc.Text())); err != nil {
			c.send(err.Error() + "\n")
			continue
		}
		registered = true
	}

	// Messages are handled starting from the next commit; discard for now.
	for sc.Scan() {
	}
	if err := sc.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("read %s: %v", c.name, err)
	}
}

func prompt(name string) string {
	return fmt.Sprintf("[%s][%s]:", time.Now().Format(timeLayout), name)
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

func (s *Server) join(c *client, name string) error {
	if err := validName(name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for o := range s.clients {
		if o.name == name {
			return errors.New("Name already taken.")
		}
	}
	c.name = name
	for _, h := range s.history {
		c.send(h + "\n")
	}
	s.clients[c] = struct{}{}
	s.announce(c, name+" has joined our chat...")
	c.send(prompt(name))
	return nil
}

func (s *Server) leave(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[c]; !ok {
		return
	}
	delete(s.clients, c)
	s.announce(c, c.name+" has left our chat...")
}

func (s *Server) announce(skip *client, text string) {
	for o := range s.clients {
		if o != skip {
			o.send("\n" + text + "\n" + prompt(o.name))
		}
	}
}