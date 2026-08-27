//go:build !linux

package tcp

import (
	"errors"
	"net"
)

// OriginalDst is only implemented on Linux, where the connection tracking keeps
// the address the client dialed before the host translated it.
func OriginalDst(_ *net.TCPConn) (net.Addr, error) {
	return nil, errors.New("recovering the original destination is only supported on Linux")
}
