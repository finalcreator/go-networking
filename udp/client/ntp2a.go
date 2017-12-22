package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

// This example of the NTP client uses unix datagram
// instead of the user datagrap protocol (udp) to communicate
// to a time server. Obviously, the server must use unix datagram
// as well.
func main() {
	var path string
	flag.StringVar(&path, "p", "/tmp/time.sock", "NTP client sock endpoint")
	flag.Parse()

	// req data packet is a 48-byte long value
	// that is used for sending time request.
	req := make([]byte, 48)

	// req is initialized with 0x1B or 0001 1011 which is
	// a request setting for time server.
	req[0] = 0x1B

	// rsp byte slice used to receive server response
	rsp := make([]byte, 48)

	// create UDPAddr
	raddr, err := net.ResolveUnixAddr("unixgram", path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	laddr := &net.UnixAddr{Name: fmt.Sprintf("%s-client", raddr.Name), Net: "unixgram"}
	// setup connection socket
	conn, err := net.DialUnix("unixgram", laddr, raddr)
	if err != nil {
		fmt.Printf("failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println("failed while closing connection:", err)
		}
	}()

	fmt.Printf("ntp time from (uinxgram) (%s)\n", conn.RemoteAddr())

	// send time request
	// write packet ntp to IO
	if _, err = conn.Write(req); err != nil {
		fmt.Printf("failed to send request: %v\n", err)
		os.Exit(1)
	}

	// now, block to receive server response
	read, err := conn.Read(rsp)
	if err != nil {
		fmt.Printf("failed to receive response: %v\n", err)
		os.Exit(1)
	}
	//ensure we read 48 bytes back (NTP protocol spec)
	if read != 48 {
		fmt.Println("did not get all expected bytes from server")
		os.Exit(1)
	}

	// NTP data comes in as big-endian (LSB [0...47] MSB)
	// with a 64-bit value containing the server time in seconds
	// where the first 32-bits are seconds and last 32-bit are fractional.
	// The following extracts the seconds from [0...[40:43]...47]
	// it is the number of secs since 1900 (NTP epoch)
	secs := binary.BigEndian.Uint32(rsp[40:])
	frac := binary.BigEndian.Uint32(rsp[44:])

	// Many OSs use Unix time epoch which is num of secs since 1970,
	// while NTP's ephoch starts on Jan 1, 1900.  Therefore,
	// to get the correct time, we must adjust the epocs properly
	// by removing 70 yrs of seconds (1970-1900) offset.
	ntpEpoch := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	unixEpoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	offset := unixEpoch.Sub(ntpEpoch).Seconds()
	now := float64(secs) - offset
	fmt.Printf("%v\n", time.Unix(int64(now), int64(frac)))
}
