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

func Tunnel(localPort int, remoteHost Endpoint, bastionHost Endpoint) error {
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

func forward(remoteHost, bastionHost Endpoint, localConn net.Conn) {
	sshConfig, err := getSSHConfig(bastionHost)
	if err != nil {
		log.Error(err)
	}

	serverConn, err := ssh.Dial("tcp", bastionHost.String(), sshConfig)
	if err != nil {
		log.Errorf("server dial error: %s", err)
		return
	}
	log.Debugf("connected to %s (1 of 2)\n", bastionHost.String())
	remoteConn, err := serverConn.Dial("tcp", remoteHost.String())
	if err != nil {
		log.Errorf("remote dial error: %s", err)
		return
	}
	log.Debugf("connected to %s (2 of 2)\n", remoteHost.String())
	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			log.Errorf("io.Copy error: %s", err)
		}
	}
	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func Connect(bastionHost Endpoint) error {

	sshConfig, err := getSSHConfig(bastionHost)
	if err != nil {
		return nil
	}

	client, err := ssh.Dial("tcp", bastionHost.String(), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial: %s", err)
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

func getSSHConfig(ep Endpoint) (*ssh.ClientConfig, error) {
	key, err := ssh.ParsePrivateKey([]byte(ep.PrivateKey))
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: ep.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}
