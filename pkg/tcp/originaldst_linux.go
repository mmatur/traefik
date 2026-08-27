package tcp

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"

	"golang.org/x/sys/unix"
)

// ip6tSOOriginalDst is the IPv6 counterpart of unix.SO_ORIGINAL_DST, read at
// the SOL_IPV6 level. golang.org/x/sys does not define it.
const ip6tSOOriginalDst = 80

// OriginalDst returns the address the client dialed before the host translated
// it, as kept by the connection tracking. A proxy sitting behind a destination
// NAT, such as a Kubernetes Service, has no other way to know which of the
// addresses it answers for the connection was sent to.
func OriginalDst(conn *net.TCPConn) (net.Addr, error) {
	localAddr, ok := conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected local address type %T", conn.LocalAddr())
	}

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("getting raw connection: %w", err)
	}

	var (
		addrPort netip.AddrPort
		sockErr  error
	)

	// A connection accepted on a dual stack listener holds an IPv4-mapped
	// address, and is tracked by the IPv4 connection tracking.
	ipv4 := localAddr.IP.To4() != nil

	if err := rawConn.Control(func(fd uintptr) {
		if ipv4 {
			addrPort, sockErr = originalDst4(int(fd))
			return
		}

		addrPort, sockErr = originalDst6(int(fd))
	}); err != nil {
		return nil, fmt.Errorf("controlling raw connection: %w", err)
	}

	if sockErr != nil {
		return nil, fmt.Errorf("getting original destination: %w", sockErr)
	}

	return net.TCPAddrFromAddrPort(addrPort), nil
}

// originalDst4 reads the IPv4 original destination. The option value is a
// sockaddr_in, which the IPv6Mreq buffer is large enough to hold.
func originalDst4(fd int) (netip.AddrPort, error) {
	mreq, err := unix.GetsockoptIPv6Mreq(fd, unix.SOL_IP, unix.SO_ORIGINAL_DST)
	if err != nil {
		return netip.AddrPort{}, err
	}

	// struct sockaddr_in: 2 bytes of family, 2 bytes of port, 4 bytes of address.
	port := binary.BigEndian.Uint16(mreq.Multiaddr[2:4])
	addr := netip.AddrFrom4([4]byte(mreq.Multiaddr[4:8]))

	return netip.AddrPortFrom(addr, port), nil
}

// originalDst6 reads the IPv6 original destination. The option value is a
// sockaddr_in6, which the IPv6MTUInfo buffer starts with.
func originalDst6(fd int) (netip.AddrPort, error) {
	info, err := unix.GetsockoptIPv6MTUInfo(fd, unix.SOL_IPV6, ip6tSOOriginalDst)
	if err != nil {
		return netip.AddrPort{}, err
	}

	// The kernel writes the port in network order, which the decoding of the
	// struct into a host order integer has already swapped on a little endian
	// architecture.
	var portBytes [2]byte
	binary.NativeEndian.PutUint16(portBytes[:], info.Addr.Port)

	addr := netip.AddrFrom16(info.Addr.Addr)

	return netip.AddrPortFrom(addr, binary.BigEndian.Uint16(portBytes[:])), nil
}
