/*
Package unix contains a work-around for representing http-over-Unix-sockets
using Go's net/http.

Paths to sockets can be relative or absolute, but relative paths must start with
a ".". If the first character is not "/" or ".", it is assumed to be a TCP host.

To encode a path, all "/" characters are replaced by "-"s. Because of this,
dashes are disallowed in UNIX path names. Paths are required to match the
regular expression:

	^[/a-zA-Z0-9.]*$
*/
package unix

import (
	"errors"
	"net"
	"regexp"
	"strings"
)

var unix *regexp.Regexp = regexp.MustCompile("^[/a-zA-Z0-9\\.]*$")

func Dialer(_, encoded string) (net.Conn, error) {
	decoded := Decode(encoded)
	return net.Dial(Network(decoded), decoded)
}

func Network(addr string) string {
	if addr[0] == '/' || addr[0] == '.' {
		return "unix"
	} else {
		return "tcp"
	}
}

func Encode(addr string) (string, error) {
	switch Network(addr) {
	case "unix":
		if !unix.MatchString(addr) {
			return "", errors.New("Invalid address path " + addr + " (must contain only dots, slashes, and alphanumeric characters, due to the way we're hacking HTTP-over-Unix-sockets into Go)")
		}
		addr = strings.Replace(addr, "/", "-", -1)
	case "tcp":
		if addr[0] == '-' {
			return "", errors.New("Invalid address " + addr + " (cannot begin with a -, due to the way we're hacking HTTP-over-Unix-sockets into Go)")
		}
	}

	return "http://" + addr, nil
}

func Decode(addr string) string {
	// Nuke the http:// if needed (may be removed by the HTTP
	// library)
	addr = strings.TrimPrefix(addr, "http://")

	if addr[0] == '-' || addr[0] == '.' {
		// Unix address
		// Remove a port, if it's been added by the HTTP library
		addr = strings.SplitN(addr, ":", 2)[0]

		// Actually decode
		addr = strings.Replace(addr, "-", "/", -1)
	}

	return addr
}
