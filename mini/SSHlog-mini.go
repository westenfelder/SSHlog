package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/gliderlabs/ssh"
)

// command line argument variables
var keyFlag string
var portFlag int

func main() {

	// command line arguments
	flag.StringVar(&keyFlag, "k", "/etc/ssh/ssh_host_ed25519_key", "server private key")
	flag.IntVar(&portFlag, "p", 22, "port")
	flag.Parse()

	// use server's private key
	hostKeyFile := ssh.HostKeyFile(keyFlag)

	// capture password
	passwordAuth := ssh.PasswordAuth(func(ctx ssh.Context, pass string) bool {
		fmt.Println("LOGIN ATTEMPT", "  Address:", ctx.RemoteAddr().String(), "  Username:", ctx.User(), "  Password:", pass)
		
		// always return false to prevent login
		return false
	})

	fmt.Println("STARTING SSHLOG")

	// start server
	startErr := ssh.ListenAndServe(":"+strconv.Itoa(portFlag), nil, hostKeyFile, passwordAuth)
	if startErr != nil {
		fmt.Println("FAILED TO START SSHLOG\t", startErr.Error())
	}

}