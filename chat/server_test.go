package chat

import (
	"net"
	"strings"
	"testing"
	"time"
)

func startServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go NewServer().Serve(ln)
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

type tc struct {
	t    *testing.T
	conn net.Conn
	buf  string
}

func dial(t *testing.T, addr string) *tc {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return &tc{t: t, conn: conn}
}

// expect reads until substr shows up in the unread data.
func (c *tc) expect(substr string) {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	b := make([]byte, 1024)
	for !strings.Contains(c.buf, substr) {
		n, err := c.conn.Read(b)
		if err != nil {
			c.t.Fatalf("waiting for %q, got %q: %v", substr, c.buf, err)
		}
		c.buf += string(b[:n])
	}
	c.buf = c.buf[strings.Index(c.buf, substr)+len(substr):]
}

func (c *tc) say(s string) { c.conn.Write([]byte(s + "\n")) }

func join(t *testing.T, addr, name string) *tc {
	c := dial(t, addr)
	c.expect("[ENTER YOUR NAME]: ")
	c.say(name)
	c.expect("[")
	return c
}

func TestLogoAndName(t *testing.T) {
	c := dial(t, startServer(t))
	c.expect("Welcome to TCP-Chat!")
	c.expect("qKKb")
	c.expect("[ENTER YOUR NAME]: ")
}

func TestInvalidNames(t *testing.T) {
	c := dial(t, startServer(t))
	c.expect("[ENTER YOUR NAME]: ")

	c.say("")
	c.expect("Name cannot be empty.")
	c.expect("[ENTER YOUR NAME]: ")

	c.say(strings.Repeat("x", maxNameLen+1))
	c.expect("Name too long")
	c.expect("[ENTER YOUR NAME]: ")

	c.say("a\x01b")
	c.expect("Name contains invalid characters.")
	c.expect("[ENTER YOUR NAME]: ")
}

func TestDuplicateName(t *testing.T) {
	addr := startServer(t)
	join(t, addr, "a")
	c := dial(t, addr)
	c.expect("[ENTER YOUR NAME]: ")
	c.say("a")
	c.expect("Name already taken.")
}

func TestJoinLeaveNotices(t *testing.T) {
	addr := startServer(t)
	a := join(t, addr, "alice")
	b := join(t, addr, "bob")
	a.expect("bob has joined our chat...")

	c := join(t, addr, "carol")
	a.expect("carol has joined our chat...")
	b.expect("carol has joined our chat...")

	b.conn.Close()
	a.expect("bob has left our chat...")
	c.expect("bob has left our chat...")
}

func TestMaxConnections(t *testing.T) {
	addr := startServer(t)
	for i := 0; i < maxClients; i++ {
		join(t, addr, string(rune('a'+i)))
	}
	c := dial(t, addr)
	c.expect("Chat is full")
}