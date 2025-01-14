package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/k1LoW/sshc"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// SSHClient ...
type SSHClient struct {
	host     string
	path     string
	client   *ssh.Client
	lineChan chan Line
	logger   *zap.Logger
}

// NewSSHClient ...
func NewSSHClient(l *zap.Logger, host string, user string, port int, path string, passphrase []byte) (Client, error) {
	options := []sshc.Option{}
	if user != "" {
		options = append(options, sshc.User(user))
	}
	if port > 0 {
		options = append(options, sshc.Port(port))
	}
	options = append(options, sshc.Passphrase(passphrase))

	client, err := sshc.NewClient(host, options...)
	if err != nil {
		return nil, err
	}
	return &SSHClient{
		client:   client,
		host:     host,
		path:     path,
		lineChan: make(chan Line),
		logger:   l,
	}, nil
}

// Read ...
func (c *SSHClient) Read(ctx context.Context, st *time.Time, et *time.Time) error {
	cmd := buildReadCommand(c.path, st)
	return c.Exec(ctx, cmd)
}

// Tailf ...
func (c *SSHClient) Tailf(ctx context.Context) error {
	cmd := buildTailfCommand(c.path)
	return c.Exec(ctx, cmd)
}

// Ls ...
func (c *SSHClient) Ls(ctx context.Context, st *time.Time, et *time.Time) error {
	cmd := buildLsCommand(c.path, st)
	return c.Exec(ctx, cmd)
}

// Copy ...
func (c *SSHClient) Copy(ctx context.Context, filePath string, dstDir string) error {
	dstLogFilePath := filepath.Join(dstDir, c.host, filePath)
	dstLogDir := filepath.Dir(dstLogFilePath)
	err := os.MkdirAll(dstLogDir, 0755)
	if err != nil {
		return err
	}
	catCmd := fmt.Sprintf("ssh %s sudo cat %s > %s", c.host, filePath, dstLogFilePath)
	cmd := exec.CommandContext(ctx, "sh", "-c", catCmd)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// RandomOne ...
func (c *SSHClient) RandomOne(ctx context.Context) error {
	cmd := buildRandomOneCommand(c.path)

	return c.Exec(ctx, cmd)
}

// Exec ...
func (c *SSHClient) Exec(ctx context.Context, cmd string) error {
	defer close(c.lineChan)
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	c.logger.Info("Create new SSH session")
	defer session.Close()

	var tzOut []byte
	err = func() error {
		session, err := c.client.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()
		tzCmd := `date +"%z"`
		tzOut, err = session.Output(tzCmd)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	// FIXME
	// _, err = session.StderrPipe()
	// if err != nil {
	// 	return err
	// }

	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go bindReaderAndChan(innerCtx, cancel, c.logger, &stdout, c.lineChan, c.host, c.path, strings.TrimRight(string(tzOut), "\n"))

	err = session.Start(cmd)
	if err != nil {
		return err
	}

	go func() {
		<-innerCtx.Done()
		session.Close()
		c.logger.Info("Close SSH session")
	}()

	err = session.Wait()
	if err != nil {
		return err
	}

	<-innerCtx.Done()

	return nil
}

// Out ...
func (c *SSHClient) Out() <-chan Line {
	return c.lineChan
}
