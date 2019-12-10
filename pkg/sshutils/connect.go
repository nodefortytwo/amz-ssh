package sshutils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"

	"io"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

func Tunnel(localPort int, remoteHost EndpointIface, bastionHost EndpointIface) error {
	log.Debug("Opening tunnel")

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "localhost", localPort))
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Infof("listening on %v", listener.Addr().(*net.TCPAddr))
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		log.Debug("accepted connection")
		go forward(remoteHost, bastionHost, conn)
	}
}

func forward(remoteHost, bastionEndpoint EndpointIface, localConn net.Conn) {
	sshConfig, err := bastionEndpoint.GetSSHConfig()
	if err != nil {
		log.Error(err)
	}

	serverConn, err := ssh.Dial("tcp", bastionEndpoint.String(), sshConfig)
	if err != nil {
		log.Errorf("server dial error: %s", err)
		return
	}
	log.Debugf("connected to %s (1 of 2)", bastionEndpoint.String())

	remoteConn, err := serverConn.Dial("tcp", remoteHost.String())
	if err != nil {
		log.Errorf("remote dial error: %s", err)
		return
	}
	log.Debugf("connected to %s (2 of 2)", remoteHost.String())

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			log.Errorf("io.Copy error: %s", err)
		}
	}
	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func Connect(bastionEndpoints ...EndpointIface) error {

	var client *ssh.Client
	for _, bastionEndpoint := range bastionEndpoints {
		sshConfig, err := bastionEndpoint.GetSSHConfig()
		if err != nil {
			return nil
		}

		serviceAddr := bastionEndpoint.String()
		log.Infof("Attempting to connect to %s", serviceAddr)
		// Tf this is the first endpint in the chain, create a new client
		// Otherwise use the previous ssh client
		if client == nil {
			client, err = ssh.Dial("tcp", serviceAddr, sshConfig)
			if err != nil {
				return fmt.Errorf("failed to dial: %s", err)
			}
		} else {
			conn, err := client.Dial("tcp", serviceAddr)
			if err != nil {
				return fmt.Errorf("failed to dial: %s", err)
			}
			ncc, chans, reqs, err := ssh.NewClientConn(conn, serviceAddr, sshConfig)
			if err != nil {
				return fmt.Errorf("failed to create new ssh connection to %s: %s", serviceAddr, err)
			}
			client = ssh.NewClient(ncc, chans, reqs)
		}
	}

	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create new session: %s", err)
	}
	defer sess.Close()

	// Set IO
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	sess.Stdin = os.Stdin

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	fileDescriptor := int(os.Stdin.Fd())

	if terminal.IsTerminal(fileDescriptor) {
		originalState, err := terminal.MakeRaw(fileDescriptor)
		if err != nil {
			return nil
		}
		defer terminal.Restore(fileDescriptor, originalState)

		termWidth, termHeight, err := terminal.GetSize(fileDescriptor)
		if err != nil {
			return err
		}

		err = sess.RequestPty("xterm-256color", termHeight, termWidth, modes)
		if err != nil {
			return err
		}
	}

	if err := sess.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}

	return sess.Wait()
}
