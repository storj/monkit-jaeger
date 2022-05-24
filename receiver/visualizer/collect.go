// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/zeebo/errs"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

func Collect(ctx context.Context, addr string, id int64, cb func(*jaeger.Span) error) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return errs.Wrap(err)
	}

	scli, err := sshDial(host)
	if err != nil {
		return errs.Wrap(err)
	}
	defer scli.Close()

	hcli := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			Dial:                  scli.Dial,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	resp, err := hcli.Get(fmt.Sprintf("http://localhost:%s/%d", port, id))
	if err != nil {
		return errs.Wrap(err)
	}
	defer resp.Body.Close()

	tport := thrift.NewStreamTransport(resp.Body, nil)
	tproto := thrift.NewTCompactProtocol(tport)

	for {
		var span jaeger.Span
		if err := span.Read(tproto); err != nil {
			return errs.Wrap(err)
		}
		if err := cb(&span); err != nil {
			return errs.Wrap(err)
		}
		if err := ctx.Err(); err != nil {
			return errs.Wrap(err)
		}
	}
}

func sshDial(host string) (cli *ssh.Client, err error) {
	var closers []io.Closer
	defer func() {
		if err != nil {
			for _, c := range closers {
				_ = c.Close()
			}
		}
	}()

	var cfg ssh.ClientConfig

	us, err := user.Current()
	if err != nil {
		return nil, errs.Wrap(err)
	}
	cfg.User = us.Username

	kh, err := knownhosts.New(filepath.Join(us.HomeDir, ".ssh/known_hosts"))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	cfg.HostKeyCallback = kh

	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return nil, errs.New("must have an SSH_AUTH_SOCK env var set")
	}

	agentConn, err := net.Dial("unix", authSock)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	closers = append(closers, agentConn)

	agent := agent.NewClient(agentConn)
	cfg.Auth = append(cfg.Auth, ssh.PublicKeysCallback(agent.Signers))

	cli, err = ssh.Dial("tcp", host+":22", &cfg)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return cli, nil
}
