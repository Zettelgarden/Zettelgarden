package mail

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSMTPServer is a tiny in-process SMTP server used to exercise the real
// SMTP code path in sendMailImpl. It speaks just enough SMTP (220 greeting,
// EHLO->250, optional AUTH->235, MAIL FROM->250, RCPT TO->250, DATA->354 with
// dot-terminated body capture, QUIT) to satisfy the go-mail client, and it can
// upgrade to TLS on STARTTLS with a self-signed certificate.
type fakeSMTPServer struct {
	mu       sync.Mutex
	messages [][]byte // captured DATA payloads
	commands []string // every command line received
	authed   bool     // true once an AUTH exchange succeeded
}

// startFakeSMTPServer listens on 127.0.0.1:0 and returns the server plus its
// address. withTLS makes the server advertise and accept STARTTLS.
func startFakeSMTPServer(t *testing.T, withTLS bool) (*fakeSMTPServer, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fake smtp listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	var cert tls.Certificate
	if withTLS {
		cert = selfSignedCert(t)
	}

	srv := &fakeSMTPServer{}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go srv.serve(conn, withTLS, cert)
		}
	}()
	return srv, ln.Addr().String()
}

func (s *fakeSMTPServer) serve(conn net.Conn, withTLS bool, cert tls.Certificate) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	write := func(line string) bool {
		if _, err := fmt.Fprintf(conn, "%s\r\n", line); err != nil {
			return false
		}
		return true
	}

	if !write("220 fake-smtp ESMTP ready") {
		return
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmdLine := strings.TrimSpace(line)
		upper := strings.ToUpper(cmdLine)

		s.mu.Lock()
		s.commands = append(s.commands, cmdLine)
		s.mu.Unlock()

		switch {
		case strings.HasPrefix(upper, "EHLO"):
			caps := []string{"250-fake-smtp greets you", "250-8BITMIME"}
			if withTLS {
				caps = append(caps, "250-STARTTLS")
			}
			caps = append(caps, "250 AUTH PLAIN LOGIN")
			for _, c := range caps {
				if !write(c) {
					return
				}
			}
		case strings.HasPrefix(upper, "STARTTLS"):
			if !write("220 2.0.0 Ready to start TLS") {
				return
			}
			tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}})
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
		case strings.HasPrefix(upper, "AUTH"):
			rest := strings.TrimSpace(cmdLine[len("AUTH"):])
			if len(strings.Fields(rest)) <= 1 {
				// challenge-response form: server sends 334, client replies
				if !write("334 ") {
					return
				}
				if _, err := reader.ReadString('\n'); err != nil {
					return
				}
			}
			s.mu.Lock()
			s.authed = true
			s.mu.Unlock()
			if !write("235 2.7.0 Authentication successful") {
				return
			}
		case strings.HasPrefix(upper, "MAIL FROM"):
			if !write("250 2.1.0 Ok") {
				return
			}
		case strings.HasPrefix(upper, "RCPT TO"):
			if !write("250 2.1.5 Ok") {
				return
			}
		case strings.HasPrefix(upper, "DATA"):
			if !write("354 End data with <CR><LF>.<CR><LF>") {
				return
			}
			var buf bytes.Buffer
			for {
				dl, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
				buf.WriteString(dl)
			}
			s.mu.Lock()
			s.messages = append(s.messages, buf.Bytes())
			s.mu.Unlock()
			if !write("250 2.0.0 Ok: queued") {
				return
			}
		case strings.HasPrefix(upper, "RSET"):
			if !write("250 2.0.0 Ok") {
				return
			}
		case strings.HasPrefix(upper, "NOOP"):
			if !write("250 2.0.0 Ok") {
				return
			}
		case strings.HasPrefix(upper, "QUIT"):
			write("221 2.0.0 Bye")
			return
		default:
			if !write("502 5.5.2 Command not recognized") {
				return
			}
		}
	}
}

// selfSignedCert builds a throwaway ECDSA server certificate for 127.0.0.1.
func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv}
}

// fakeSMTPClient builds a MailClient pointed at the fake server with the given
// SMTP options.
func fakeSMTPClient(addr string, mutate func(*MailClient)) *MailClient {
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	m := &MailClient{
		SMTPHost: host,
		SMTPPort: port,
		SMTPFrom: "noreply@test.com",
		StartTLS: false,
	}
	if mutate != nil {
		mutate(m)
	}
	return m
}

func assertMessageContains(t *testing.T, msg []byte, want string) {
	t.Helper()
	if !bytes.Contains(msg, []byte(want)) {
		t.Errorf("captured message missing %q:\n%s", want, msg)
	}
}

// TestSendMailImplSMTPPlainText drives a plain-text email (ASCII subject, no
// credentials) through the real SMTP transport and checks the captured message
// headers and body.
func TestSendMailImplSMTPPlainText(t *testing.T) {
	srv, addr := startFakeSMTPServer(t, false)
	m := fakeSMTPClient(addr, nil)

	err := m.sendMailImpl(Email{
		Subject:   "Hello",
		Recipient: "user@example.com",
		Body:      "Hello plain body",
		IsHTML:    false,
	})
	if err != nil {
		t.Fatalf("sendMailImpl failed: %v", err)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.messages) != 1 {
		t.Fatalf("expected 1 captured message, got %d", len(srv.messages))
	}
	msg := srv.messages[0]
	assertMessageContains(t, msg, `From: "Zettelgarden" <noreply@test.com>`)
	assertMessageContains(t, msg, "To: <user@example.com>")
	assertMessageContains(t, msg, "Subject: Hello")
	assertMessageContains(t, msg, "Content-Type: text/plain; charset=UTF-8")
	assertMessageContains(t, msg, "Hello plain body")
	if srv.authed {
		t.Error("expected no AUTH exchange without credentials")
	}
}

// TestSendMailImplSMTPHTMLUnicodeSubject drives an HTML email with a non-ASCII
// subject (emoji) and asserts the Subject header is RFC 2047 encoded and the
// body is HTML.
func TestSendMailImplSMTPHTMLUnicodeSubject(t *testing.T) {
	srv, addr := startFakeSMTPServer(t, false)
	m := fakeSMTPClient(addr, nil)

	err := m.sendMailImpl(Email{
		Subject:   "Welcome to Zettelgarden! 🌱",
		Recipient: "user@example.com",
		Body:      "<b>Happy gardening!</b>",
		IsHTML:    true,
	})
	if err != nil {
		t.Fatalf("sendMailImpl failed: %v", err)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.messages) != 1 {
		t.Fatalf("expected 1 captured message, got %d", len(srv.messages))
	}
	msg := srv.messages[0]
	// Non-ASCII subject must be RFC 2047 encoded (contains =?UTF-8?).
	assertMessageContains(t, msg, "Subject: =?UTF-8?")
	assertMessageContains(t, msg, "Content-Type: text/html; charset=UTF-8")
	assertMessageContains(t, msg, "<b>Happy gardening!</b>")
}

// TestSendMailImplSMTPAuth verifies credentials are used: the client issues an
// AUTH command and the server marks the exchange authenticated.
func TestSendMailImplSMTPAuth(t *testing.T) {
	srv, addr := startFakeSMTPServer(t, false)
	m := fakeSMTPClient(addr, func(m *MailClient) {
		m.SMTPUsername = "smtp-user"
		m.SMTPPassword = "smtp-pass"
	})

	err := m.sendMailImpl(Email{
		Subject:   "Auth test",
		Recipient: "user@example.com",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("sendMailImpl failed: %v", err)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if !srv.authed {
		t.Fatal("expected an AUTH exchange when credentials are configured")
	}
	found := false
	for _, c := range srv.commands {
		if strings.HasPrefix(strings.ToUpper(c), "AUTH") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected an AUTH command, got commands: %v", srv.commands)
	}
}

// TestSendMailImplSMTPStartTLS verifies the default production path: STARTTLS
// upgrade with a (self-signed, skipped-verify in test) certificate.
func TestSendMailImplSMTPStartTLS(t *testing.T) {
	srv, addr := startFakeSMTPServer(t, true)
	m := fakeSMTPClient(addr, func(m *MailClient) {
		m.StartTLS = true
		m.insecureSkipTLSVerify = true
	})

	err := m.sendMailImpl(Email{
		Subject:   "TLS test",
		Recipient: "user@example.com",
		Body:      "over tls",
	})
	if err != nil {
		t.Fatalf("sendMailImpl failed: %v", err)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.messages) != 1 {
		t.Fatalf("expected 1 captured message, got %d", len(srv.messages))
	}
	assertMessageContains(t, srv.messages[0], "Subject: TLS test")
	assertMessageContains(t, srv.messages[0], "over tls")
	found := false
	for _, c := range srv.commands {
		if strings.HasPrefix(strings.ToUpper(c), "STARTTLS") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected STARTTLS command, got commands: %v", srv.commands)
	}
}
