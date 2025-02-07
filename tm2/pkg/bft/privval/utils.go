package privval

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/gnolang/gno/tm2/pkg/errors"
	osm "github.com/gnolang/gno/tm2/pkg/os"
)

// IsConnTimeout returns a boolean indicating whether the error is known to
// report that a connection timeout occurred. This detects both fundamental
// network timeouts, as well as ErrConnTimeout errors.
func IsConnTimeout(err error) bool {
	switch errors.Cause(err).(type) {
	case EndpointTimeoutError:
		return true
	case timeoutError:
		return true
	default:
		return false
	}
}

// NewSignerListener creates a new SignerListenerEndpoint using the corresponding listen address
func NewSignerListener(listenAddr string, logger *slog.Logger) (_ *SignerListenerEndpoint, rerr error) {
	var listener net.Listener

	protocol, address := osm.ProtocolAndAddress(listenAddr)
	ln, err := net.Listen(protocol, address)
	if err != nil {
		return nil, err
	}

	defer func() {
		if rerr != nil {
			ln.Close()
		}
	}()

	switch protocol {
	case "unix":
		listener = NewUnixListener(ln)
	case "tcp":
		// TODO: persist this key so external signer can actually authenticate us
		listener = NewTCPListener(ln, ed25519.GenPrivKey())
	default:
		return nil, fmt.Errorf(
			"wrong listen address: expected either 'tcp' or 'unix' protocols, got %s",
			protocol,
		)
	}

	pve := NewSignerListenerEndpoint(logger.With("module", "privval"), listener)

	return pve, nil
}
