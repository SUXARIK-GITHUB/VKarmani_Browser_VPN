package app

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

func socksDialContext(proxyAddr string) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, target string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, errors.New("SOCKS dialer supports TCP only")
		}
		d := net.Dialer{Timeout: 8 * time.Second}
		c, err := d.DialContext(ctx, "tcp", proxyAddr)
		if err != nil {
			return nil, err
		}
		fail := func(e error) (net.Conn, error) { c.Close(); return nil, e }
		deadline, ok := ctx.Deadline()
		if ok {
			_ = c.SetDeadline(deadline)
		} else {
			_ = c.SetDeadline(time.Now().Add(12 * time.Second))
		}
		if _, err = c.Write([]byte{0x05, 0x01, 0x00}); err != nil {
			return fail(err)
		}
		resp := make([]byte, 2)
		if _, err = io.ReadFull(c, resp); err != nil {
			return fail(err)
		}
		if resp[0] != 0x05 || resp[1] != 0x00 {
			return fail(errors.New("SOCKS authentication negotiation failed"))
		}
		host, portS, err := net.SplitHostPort(target)
		if err != nil {
			return fail(err)
		}
		port, err := strconv.Atoi(portS)
		if err != nil || port < 1 || port > 65535 {
			return fail(errors.New("invalid target port"))
		}
		buf := []byte{0x05, 0x01, 0x00}
		if ip := net.ParseIP(host); ip != nil {
			if v4 := ip.To4(); v4 != nil {
				buf = append(buf, 0x01)
				buf = append(buf, v4...)
			} else {
				buf = append(buf, 0x04)
				buf = append(buf, ip.To16()...)
			}
		} else {
			if len(host) > 255 {
				return fail(errors.New("SOCKS target hostname too long"))
			}
			buf = append(buf, 0x03, byte(len(host)))
			buf = append(buf, []byte(host)...)
		}
		p := make([]byte, 2)
		binary.BigEndian.PutUint16(p, uint16(port))
		buf = append(buf, p...)
		if _, err = c.Write(buf); err != nil {
			return fail(err)
		}
		head := make([]byte, 4)
		if _, err = io.ReadFull(c, head); err != nil {
			return fail(err)
		}
		if head[0] != 0x05 || head[1] != 0x00 {
			return fail(fmt.Errorf("SOCKS connect failed code=%d", head[1]))
		}
		switch head[3] {
		case 0x01:
			_, err = io.CopyN(io.Discard, c, 4)
		case 0x04:
			_, err = io.CopyN(io.Discard, c, 16)
		case 0x03:
			var l [1]byte
			if _, err = io.ReadFull(c, l[:]); err == nil {
				_, err = io.CopyN(io.Discard, c, int64(l[0]))
			}
		default:
			err = errors.New("invalid SOCKS address type")
		}
		if err != nil {
			return fail(err)
		}
		if _, err = io.CopyN(io.Discard, c, 2); err != nil {
			return fail(err)
		}
		_ = c.SetDeadline(time.Time{})
		return c, nil
	}
}
