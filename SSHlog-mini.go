package main

/*
#cgo LDFLAGS: -lcrypt
#include <crypt.h>
*/
import "C"

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

// command line argument variables
var silentFlag bool
var keyFlag string
var loginFlag bool
var portFlag int
var verboseFlag bool
var messageFlag string

// Adjust window size
func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

// Parse /etc/shadow file and lookup password hash for given username
func findHash(username string) string {
	// Parse /etc/shadow file to 2d array
	shadow_file, _ := os.ReadFile("/etc/shadow")
	shadow_string := string(shadow_file)
	shadow_lines := strings.Split(shadow_string, "\n")
	var shadow_entries [][]string
	for i := 0; i < len(shadow_lines); i++ {
		shadow_entries = append(shadow_entries, strings.Split(shadow_lines[i], ":"))
	}
	shadow_entries = shadow_entries[:len(shadow_entries)-1]

	// grab hash from array
	var hash string = ""
	for i := 0; i < len(shadow_entries); i++ {
		if shadow_entries[i][0] == username {
			hash = shadow_entries[i][1]
		}
	}

	return hash
}

// Read file character by character to buffer
func readChar(r io.Reader) (bytes.Buffer, error) {
	var buffer bytes.Buffer
	var err error
	singleByteBuffer := []byte{0}

	// read single byte, return err if EOF
	_, err = r.Read(singleByteBuffer)
	if err != nil {
		return buffer, io.EOF
	} else {
		buffer.Write(singleByteBuffer)
		return buffer, nil
	}
}

func main() {

	ssh.Handle(func(s ssh.Session) {

		fmt.Println("CLIENT CONNECTED\t", "Address:", s.RemoteAddr().String())

		// start bash in users home directory
		cmdString := "cd $HOME; bash"
        cmd := exec.Command("bash", "-c", cmdString)

		// Configure pseudoterminal
		_, winCh, _ := s.Pty()

		// run start command
		f, errBash := pty.Start(cmd)
		if errBash != nil {
			fmt.Println("FAILED TO START BASH\t", errBash.Error())
		}

		//Adjust window size
		go func() {
			for win := range winCh {
				setWinsize(f, win.Width, win.Height)
			}
		}()

		// stdin
		go func() {
			io.Copy(f, s)
		}()

		// stdout
		go func() {
			// create log file
			logFileName := ".ClientLog:" + strings.TrimSpace(time.Now().Format(time.RFC3339)) + ":" + s.RemoteAddr().String()
			logFile, err := os.Create(logFileName)
			if err == nil {
				fmt.Println("CREATED LOG FILE\t", "Filename:", logFileName)
			} else {
				fmt.Println("LOG FILE CREATION FAILED\t", err.Error())
			}
			

			var err2 error
			var buffer bytes.Buffer
			for err2 == nil {
				// read from ssh stdout
				buffer, err2 = readChar(f)
				// log output
				_, writeErr := logFile.Write(buffer.Bytes())
				if writeErr != nil {
					fmt.Println("FAILED TO WRITE LOG FILE\t", writeErr.Error())
				}
				if verboseFlag {
					fmt.Print(buffer.String())
				}
				// write output to client stdout
				s.Write(buffer.Bytes())
			}
			closeErr := logFile.Close()
			if closeErr != nil {
				fmt.Println("FAILED TO CLOSE LOG FILE\t", closeErr.Error())
			}
		}()

		// wait for bash to exit
		cmd.Wait()
		// write newline on exit
		s.Write([]byte{byte('\n')})
		// write message to client
		if messageFlag != "" {
			_, messageErr := s.Write([]byte(messageFlag + "\n"))
			if messageErr != nil {
				fmt.Println("FAILED TO WRITE EXIT MESSAGE\t", messageErr.Error())
			}
		}
		// client disconnect
		fmt.Println("CLIENT DISCONNECTED\t", "Address:", s.RemoteAddr().String())

	})

	// START OF MAIN ===============================================================

	// Command line arguments
	flag.BoolVar(&silentFlag, "s", false, "silent mode")
	flag.StringVar(&keyFlag, "k", "/etc/ssh/ssh_host_ed25519_key", "server private key")
	flag.BoolVar(&loginFlag, "l", false, "prevent client login")
	flag.IntVar(&portFlag, "p", 22, "port")
	flag.BoolVar(&verboseFlag, "v", false, "log to stdout (NOT RECOMMENDED)")
	flag.StringVar(&messageFlag, "m", "", "send message to client on exit")
	flag.Parse()

	// use server's private key
	hostKeyFile := ssh.HostKeyFile(keyFlag)

	// enable password login
	passwordAuth := ssh.PasswordAuth(func(ctx ssh.Context, pass string) bool {

		fmt.Println("LOGIN ATTEMPT\t\t", "Address:", ctx.RemoteAddr().String(), "  Client:", ctx.ClientVersion(), "  Username:", ctx.User(), "  Password:", pass)

		if loginFlag {
			return false
		} else {
			// check if password is valid
			hash := findHash(ctx.User())
			// case where username is not in /etc/shadow
			if hash == "" {
				return false
			} else {
				new_hash := C.GoString(C.crypt(C.CString(pass), C.CString(hash)))
				return hash == new_hash
			}
		}
	})

	fmt.Println("STARTING SSHLOG")
	fmt.Println("CREATED LOG FILE\t", "Filename: .ServerLog")

	// start server
	startErr := ssh.ListenAndServe(":"+strconv.Itoa(portFlag), nil, hostKeyFile, passwordAuth)
	if startErr != nil {
		fmt.Println("FAILED TO START SSHLOG\t", startErr.Error())
	}

}
