package ssh

import (
	"io"
	"net"
	"reflect"
	"strconv"

	"github.com/gruntwork-io/terratest/modules/collections"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/testing"
	"golang.org/x/crypto/ssh"
)

// SshConnectionOptions are the options for an SSH connection.
type SshConnectionOptions struct {
	Username    string
	Address     string
	Port        int
	AuthMethods []ssh.AuthMethod
	Command     string
	JumpHost    *SshConnectionOptions
}

// ConnectionString returns the connection string for an SSH connection.
func (options *SshConnectionOptions) ConnectionString() string {
	return net.JoinHostPort(options.Address, strconv.Itoa(options.Port))
}

// SshSession is a container object for all resources created by an SSH session. The reason we need this is so that we can do a
// single defer in a top-level method that calls the Cleanup method to go through and ensure all of these resources are
// released and cleaned up.
type SshSession struct {
	Options  *SshConnectionOptions
	Client   *ssh.Client
	Session  *ssh.Session
	JumpHost *JumpHostSession
	Input    *func(io.WriteCloser)
}

// Cleanup cleans up an existing SSH session.
func (sshSession *SshSession) Cleanup(t testing.TestingT) {
	if sshSession == nil {
		return
	}

	// Closing the session may result in an EOF error if it's already closed (e.g. due to hitting CTRL + D), so
	// don't report those errors, as there is nothing actually wrong in that case.
	Close(t, sshSession.Session, io.EOF.Error())
	Close(t, sshSession.Client)
	sshSession.JumpHost.Cleanup(t)
}

// JumpHostSession is a session with a jump host.
type JumpHostSession struct {
	JumpHostClient        *ssh.Client
	HostVirtualConnection net.Conn
	HostConnection        ssh.Conn
}

// Cleanup cleans the jump host session up.
func (jumpHost *JumpHostSession) Cleanup(t testing.TestingT) {
	if jumpHost == nil {
		return
	}

	// Closing a connection may result in an EOF error if it's already closed (e.g. due to hitting CTRL + D), so
	// don't report those errors, as there is nothing actually wrong in that case.
	Close(t, jumpHost.HostConnection, io.EOF.Error())
	Close(t, jumpHost.HostVirtualConnection, io.EOF.Error())
	Close(t, jumpHost.JumpHostClient)
}

// Closeable can be closed.
type Closeable interface {
	Close() error
}

// Close closes a Closeable.
func Close(t testing.TestingT, closeable Closeable, ignoreErrors ...string) {
	if interfaceIsNil(closeable) {
		return
	}

	if err := closeable.Close(); err != nil && !collections.ListContains(ignoreErrors, err.Error()) {
		logger.Logf(t, "Error closing %s: %s", closeable, err.Error())
	}
}

// Go is a shitty language. Checking an interface directly against nil does not work, and if you don't know the exact
// types the interface may be ahead of time, the only way to know if you're dealing with nil is to use reflection.
// http://stackoverflow.com/questions/13476349/check-for-nil-and-nil-interface-in-go
func interfaceIsNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}
