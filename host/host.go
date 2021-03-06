package host

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jingweno/upterm/host/api/swagger/models"
	"github.com/jingweno/upterm/host/internal"
	"github.com/jingweno/upterm/upterm"
	"github.com/jingweno/upterm/utils"
	"github.com/oklog/run"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type Host struct {
	Host                   string
	SessionID              string
	KeepAlive              time.Duration
	Command                []string
	ForceCommand           []string
	Signers                []ssh.Signer
	AuthorizedKeys         []ssh.PublicKey
	AdminSocketFile        string
	SessionCreatedCallback func(*models.APIGetSessionResponse) error
	Logger                 log.FieldLogger
	Stdin                  *os.File
	Stdout                 *os.File
}

func (c *Host) createAdminSocketDir(sessionID string) (string, error) {
	uptermDir, err := utils.UptermDir()
	if err != nil {
		return "", err
	}

	socketDir := filepath.Join(uptermDir, sessionID)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return "", err
	}

	return socketDir, nil
}

func (c *Host) Run(ctx context.Context) error {
	if c.SessionID == "" {
		c.SessionID = xid.New().String()
	}
	if c.Stdin == nil {
		c.Stdin = os.Stdin
	}
	if c.Stdout == nil {
		c.Stdout = os.Stdout
	}
	if c.AdminSocketFile == "" {
		adminSocketDir, err := c.createAdminSocketDir(c.SessionID)
		if err != nil {
			return err
		}
		defer os.RemoveAll(adminSocketDir)

		c.AdminSocketFile = AdminSocketFile(adminSocketDir)
	}

	rt := internal.ReverseTunnel{
		Host:      c.Host,
		SessionID: c.SessionID,
		Signers:   c.Signers,
		KeepAlive: c.KeepAlive,
		Logger:    log.WithField("component", "reverse-tunnel"),
	}
	info, err := rt.Establish(ctx)
	if err != nil {
		return err
	}
	defer rt.Close()

	session := &models.APIGetSessionResponse{
		SessionID:    c.SessionID,
		Host:         c.Host,
		NodeAddr:     info.NodeAddr,
		Command:      c.Command,
		ForceCommand: c.ForceCommand,
	}

	if c.SessionCreatedCallback != nil {
		if err := c.SessionCreatedCallback(session); err != nil {
			return err
		}
	}

	var g run.Group
	{
		ctx, cancel := context.WithCancel(ctx)
		s := adminServer{
			Session: session,
		}
		g.Add(func() error {
			return s.Serve(ctx, c.AdminSocketFile)
		}, func(err error) {
			_ = s.Shutdown(ctx)
			cancel()
		})
	}
	{
		ctx, cancel := context.WithCancel(ctx)
		sshServer := internal.Server{
			Command:        c.Command,
			CommandEnv:     []string{fmt.Sprintf("%s=%s", upterm.HostAdminSocketEnvVar, c.AdminSocketFile)},
			ForceCommand:   c.ForceCommand,
			Signers:        c.Signers,
			AuthorizedKeys: c.AuthorizedKeys,
			Stdin:          c.Stdin,
			Stdout:         c.Stdout,
			Logger:         c.Logger,
		}
		g.Add(func() error {
			return sshServer.ServeWithContext(ctx, rt.Listener())
		}, func(err error) {
			cancel()
		})
	}

	return g.Run()
}
