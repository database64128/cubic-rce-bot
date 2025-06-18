package webhook

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/user"
	"strconv"

	"github.com/database64128/cubic-rce-bot/jsoncfg"
	"github.com/database64128/cubic-rce-bot/tslog"
)

// Config holds the configuration for the webhook server.
type Config struct {
	// Enabled controls whether the webhook server is enabled.
	Enabled bool `json:"enabled"`

	// ListenNetwork is the network to listen on (e.g., "tcp", "unix").
	ListenNetwork string `json:"listenNetwork"`

	// ListenAddress is the address to listen on (e.g., ":8080", "/run/cubic-rce-bot.sock").
	ListenAddress string `json:"listenAddress"`

	// ListenOwner optionally sets the owner of the unix domain socket.
	// It can be an integer user ID or a string username.
	ListenOwner jsoncfg.IntOrString `json:"listenOwner,omitzero"`

	// ListenGroup optionally sets the group of the unix domain socket.
	// It can be an integer group ID or a string group name.
	ListenGroup jsoncfg.IntOrString `json:"listenGroup,omitzero"`

	// ListenMode optionally sets the file mode of the unix domain socket.
	// It must be an octal number in a string (e.g., "0660").
	ListenMode jsoncfg.FileMode `json:"listenMode,omitzero"`

	// SecretToken is the optional secret token for the webhook.
	SecretToken string `json:"secretToken,omitzero"`

	// URL is the webhook URL.
	URL string `json:"url"`
}

// NewServer creates a new webhook server.
func (c *Config) NewServer(logger *tslog.Logger, handler http.Handler) *Server {
	return &Server{
		logger:  logger,
		network: c.ListenNetwork,
		server: http.Server{
			Addr:     c.ListenAddress,
			Handler:  handler,
			ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
		owner: c.ListenOwner,
		group: c.ListenGroup,
		mode:  c.ListenMode.Value(),
	}
}

// Server represents a webhook server that listens for incoming HTTP requests.
type Server struct {
	logger  *tslog.Logger
	network string
	server  http.Server
	owner   jsoncfg.IntOrString
	group   jsoncfg.IntOrString
	mode    fs.FileMode
}

// Start starts the webhook server.
func (s *Server) Start(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, s.network, s.server.Addr)
	if err != nil {
		return err
	}
	listenAddress := ln.Addr()

	if listenAddress, ok := listenAddress.(*net.UnixAddr); ok {
		if err := s.configureUnixDomainSocket(listenAddress); err != nil {
			_ = ln.Close()
			return fmt.Errorf("failed to configure unix domain socket %q: %w", listenAddress.Name, err)
		}
	}

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Failed to serve webhook", tslog.Err(err))
		}
	}()

	s.logger.Info("Started webhook server", slog.Any("listenAddress", listenAddress))
	return nil
}

func (s *Server) configureUnixDomainSocket(listenAddress *net.UnixAddr) error {
	if s.owner.IsValid() || s.group.IsValid() {
		uid, gid := -1, -1

		switch s.owner.Kind() {
		case jsoncfg.IntOrStringKindInt:
			uid = s.owner.Int()

		case jsoncfg.IntOrStringKindString:
			username := s.owner.String()
			owner, err := user.Lookup(username)
			if err != nil {
				return fmt.Errorf("failed to lookup user %q: %w", username, err)
			}

			uid, err = strconv.Atoi(owner.Uid)
			if err != nil {
				return fmt.Errorf("failed to convert user ID %q to int: %w", owner.Uid, err)
			}
		}

		switch s.group.Kind() {
		case jsoncfg.IntOrStringKindInt:
			gid = s.group.Int()

		case jsoncfg.IntOrStringKindString:
			groupName := s.group.String()
			group, err := user.LookupGroup(groupName)
			if err != nil {
				return fmt.Errorf("failed to lookup group %q: %w", groupName, err)
			}

			gid, err = strconv.Atoi(group.Gid)
			if err != nil {
				return fmt.Errorf("failed to convert group ID %q to int: %w", group.Gid, err)
			}
		}

		if err := os.Chown(listenAddress.Name, uid, gid); err != nil {
			return fmt.Errorf("failed to change ownership of %q to uid=%d, gid=%d: %w", listenAddress.Name, uid, gid, err)
		}
	}

	if s.mode != 0 {
		if err := os.Chmod(listenAddress.Name, s.mode); err != nil {
			return fmt.Errorf("failed to change mode of %q to %o: %w", listenAddress.Name, s.mode, err)
		}
	}

	return nil
}

// Stop stops the webhook server.
func (s *Server) Stop() error {
	if err := s.server.Close(); err != nil {
		return err
	}
	s.logger.Info("Stopped webhook server")
	return nil
}
