package server

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jingweno/upterm/upterm"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	ErrListnerClosed        = errors.New("routing: listener closed")
	pipeEstablishingTimeout = 10 * time.Second
)

const (
	errReadConnectionResetByPeer = "read: connection reset by peer"
	errSshDisconnectReason11     = "ssh: disconnect, reason 11:"
	errUnknownClient             = "unknown client:"
)

type FindUpstreamFunc func(conn ssh.ConnMetadata, challengeCtx ssh.AdditionalChallengeContext) (net.Conn, *ssh.AuthPipe, error)

type Routing struct {
	HostSigners      []ssh.Signer
	FindUpstreamFunc FindUpstreamFunc
	Logger           log.FieldLogger

	listener net.Listener
	mux      sync.Mutex
	doneChan chan struct{}
}

func (p *Routing) Serve(ln net.Listener) error {
	p.listener = ln

	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-p.getDoneChan():
				return ErrListnerClosed
			default:
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				p.Logger.WithError(err).Errorf("http: Accept error; retrying in %v", tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			p.Logger.WithError(err).Error("failed to accept connection")
			return err
		}

		tempDelay = 0

		logger := p.Logger.WithField("addr", conn.RemoteAddr())

		piper := &ssh.PiperConfig{
			FindUpstream:  p.FindUpstreamFunc,
			ServerVersion: upterm.ServerSSHServerVersion,
		}
		for _, signer := range p.HostSigners {
			piper.AddHostKey(signer)
		}

		go func(c net.Conn, logger log.FieldLogger) {
			defer c.Close()

			pipec := make(chan *ssh.PiperConn)
			errorc := make(chan error)

			go func() {
				p, err := ssh.NewSSHPiperConn(c, piper)

				if err != nil {
					errorc <- err
					return
				}

				pipec <- p
			}()

			var pc *ssh.PiperConn
			select {
			case pc = <-pipec:
			case err := <-errorc:
				if isIgnoredErr(err) {
					logger.WithError(err).Debug("connection establishing failed")
				} else {
					logger.WithError(err).Error("connection establishing failed")
				}
				return
			case <-time.After(pipeEstablishingTimeout):
				logger.Debug("pipe establishing timeout")
				return
			}

			defer pc.Close()

			if err := pc.Wait(); err != nil && !isIgnoredErr(err) {
				logger.WithError(err).Error("error waiting for pipe")
			}
		}(conn, logger)
	}
}

func (p *Routing) Shutdown() error {
	p.mux.Lock()
	lnerr := p.closeListenersLocked()
	p.closeDoneChanLocked()
	p.mux.Unlock()

	return lnerr
}

func (p *Routing) getDoneChan() <-chan struct{} {
	p.mux.Lock()
	defer p.mux.Unlock()

	return p.getDoneChanLocked()
}

func (p *Routing) getDoneChanLocked() chan struct{} {
	if p.doneChan == nil {
		p.doneChan = make(chan struct{})
	}

	return p.doneChan
}

func (p *Routing) closeDoneChanLocked() {
	ch := p.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by p.mux.
		close(ch)
	}
}

func (p *Routing) closeListenersLocked() error {
	return p.listener.Close()
}

func isIgnoredErr(err error) bool {
	return errors.Is(err, io.EOF) ||
		strings.Contains(err.Error(), errReadConnectionResetByPeer) ||
		strings.Contains(err.Error(), errSshDisconnectReason11) ||
		strings.Contains(err.Error(), errUnknownClient)
}
