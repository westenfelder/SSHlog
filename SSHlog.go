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
	"github.com/fatih/color"
	"github.com/gliderlabs/ssh"
)

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

// Logging function with color printing
func printLog(stringList []string, colorList []string) {

	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	white := color.New(color.FgHiWhite)
	blue := color.New(color.FgHiBlue)
	magenta := color.New(color.FgHiMagenta)

	if len(colorList) != len(stringList) {
		return
	} else {
		blue.Print(time.Now().Format(time.RFC1123), "  ")

		for i := 0; i < len(stringList); i++ {
			if colorList[i] == "red" {
				red.Print(stringList[i], " ")
			}
			if colorList[i] == "green" {
				green.Print(stringList[i], " ")
			}
			if colorList[i] == "yellow" {
				yellow.Print(stringList[i], " ")
			}
			if colorList[i] == "white" {
				white.Print(stringList[i], " ")
			}
			if colorList[i] == "blue" {
				blue.Print(stringList[i], " ")
			}
			if colorList[i] == "magenta" {
				magenta.Print(stringList[i], " ")
			}
		}

		fmt.Println()
	}
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

		if !silentFlag {
			printLog([]string{"CLIENT CONNECTED", "     Address:", s.RemoteAddr().String()}, []string{"yellow", "white", "white"})
		}

		// set start command as bash
		cmd := exec.Command("bash")

		// Configure pseudoterminal
		_, winCh, _ := s.Pty()

		// run start command
		f, errBash := pty.Start(cmd)
		if errBash != nil {
			if !silentFlag {
				printLog([]string{"FAILED TO START BASH", errBash.Error()}, []string{"red", "white"})
			}
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
			logFileName := "." + strings.TrimSpace(time.Now().Format(time.RFC3339)) + ":" + s.RemoteAddr().String()
			logFile, err := os.Create(logFileName)
			if err == nil {
				if !silentFlag {
					printLog([]string{"CREATED LOG FILE", "     Filename:", logFileName}, []string{"green", "white", "white"})
				}
			} else {
				if !silentFlag {
					printLog([]string{"LOG FILE CREATION FAILED", err.Error()}, []string{"red", "white"})
				}
			}
			

			var err2 error
			var buffer bytes.Buffer
			for err2 == nil {
				// read from ssh stdout
				buffer, err2 = readChar(f)
				// log output
				_, writeErr := logFile.Write(buffer.Bytes())
				if writeErr != nil {
					if !silentFlag{
						printLog([]string{"FAILED TO WRITE LOG FILE", writeErr.Error()}, []string{"red", "white"})
					}
				}
				if verboseFlag {
					fmt.Print(buffer.String())
				}
				// write output to client stdout
				s.Write(buffer.Bytes())
			}
			closeErr := logFile.Close()
			if closeErr != nil {
				if !silentFlag {
					printLog([]string{"FAILED TO CLOSE LOG FILE", closeErr.Error()}, []string{"red", "white"})
				}
			}
		}()

		// wait for bash to exit
		cmd.Wait()
		// write newline on exit
		s.Write([]byte{byte('\n')})
		// write message to client
		if messageFlag != "" {
			s.Write([]byte(messageFlag + "\n"))
		}
		// client disconnect
		if !silentFlag {
			printLog([]string{"CLIENT DISCONNECTED", "  Address:", s.RemoteAddr().String()}, []string{"yellow", "white", "white"})
		}

	})

	// START OF MAIN ===============================================================
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

		if !silentFlag {
			printLog([]string{"LOGIN ATTEMPT", "        Address:", ctx.RemoteAddr().String(), "  Client:", ctx.ClientVersion(), "  Username:", ctx.User(), "  Password:", pass}, []string{"red", "white", "magenta", "white", "white", "white", "magenta", "white", "magenta"})
		}

		if loginFlag {
			return false
		} else {
			// check if password is valid
			hash := findHash(ctx.User())
			new_hash := C.GoString(C.crypt(C.CString(pass), C.CString(hash)))
			return hash == new_hash
		}
	})

	if !silentFlag {
		printLog([]string{"STARTING SSHLOG"}, []string{"green"})
	}

	// start server
	startErr := ssh.ListenAndServe(":"+strconv.Itoa(portFlag), nil, hostKeyFile, passwordAuth)
	if startErr != nil {
		if !silentFlag {
			printLog([]string{"FAILED TO START SSHLOG ", startErr.Error()}, []string{"red", "white"})
		}
	}

}
