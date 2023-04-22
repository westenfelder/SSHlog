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

// Logging function with color printing
func printLog(stringList []string, colorList []string) {

	// Open or create server log file
	var serverLogFile *os.File
	var serverLogFileName string = ".ServerLog"
	serverLogFile, openErr := os.OpenFile(serverLogFileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if openErr != nil {
		serverLogFile, _ = os.Create(serverLogFileName)
	}
	
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	white := color.New(color.FgHiWhite)
	blue := color.New(color.FgHiBlue)
	magenta := color.New(color.FgHiMagenta)

	if len(colorList) != len(stringList) {
		return
	} else {
		time := time.Now().Format(time.RFC1123)
		if !silentFlag {
			blue.Print(time, "\t")
		}
		serverLogFile.WriteString(time + "\t")

		for i := 0; i < len(stringList); i++ {
			// write to server log file
			serverLogFile.WriteString(stringList[i] + " ")

			// print to stdout with color
			if colorList[i] == "red" && !silentFlag {
				red.Print(stringList[i], " ")
			}
			if colorList[i] == "green" && !silentFlag {
				green.Print(stringList[i], " ")
			}
			if colorList[i] == "yellow" && !silentFlag {
				yellow.Print(stringList[i], " ")
			}
			if colorList[i] == "white" && !silentFlag {
				white.Print(stringList[i], " ")
			}
			if colorList[i] == "blue" && !silentFlag {
				blue.Print(stringList[i], " ")
			}
			if colorList[i] == "magenta" && !silentFlag {
				magenta.Print(stringList[i], " ")
			}
		}

		// print newline
		if !silentFlag {
			fmt.Println()
		}
		serverLogFile.WriteString("\n")
	}

	// Close server log file
	serverLogFile.Close()
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

		printLog([]string{"CLIENT CONNECTED\t", "Address:", s.RemoteAddr().String()}, []string{"red", "white", "magenta"})

		// start bash in users home directory
		cmdString := "cd $HOME; bash"
        cmd := exec.Command("bash", "-c", cmdString)

		// Configure pseudoterminal
		_, winCh, _ := s.Pty()

		// run start command
		f, errBash := pty.Start(cmd)
		if errBash != nil {
			printLog([]string{"FAILED TO START BASH\t", errBash.Error()}, []string{"red", "white"})
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
				printLog([]string{"CREATED LOG FILE\t", "Filename:", logFileName}, []string{"green", "white", "white"})
			} else {
				printLog([]string{"LOG FILE CREATION FAILED\t", err.Error()}, []string{"red", "white"})
			}
			

			var err2 error
			var buffer bytes.Buffer
			for err2 == nil {
				// read from ssh stdout
				buffer, err2 = readChar(f)
				// log output
				_, writeErr := logFile.Write(buffer.Bytes())
				if writeErr != nil {
					printLog([]string{"FAILED TO WRITE LOG FILE\t", writeErr.Error()}, []string{"red", "white"})
				}
				if verboseFlag {
					fmt.Print(buffer.String())
				}
				// write output to client stdout
				s.Write(buffer.Bytes())
			}
			closeErr := logFile.Close()
			if closeErr != nil {
				printLog([]string{"FAILED TO CLOSE LOG FILE\t", closeErr.Error()}, []string{"red", "white"})
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
				printLog([]string{"FAILED TO WRITE EXIT MESSAGE\t", messageErr.Error()}, []string{"red", "white"})
			}
		}
		// client disconnect
		printLog([]string{"CLIENT DISCONNECTED\t", "Address:", s.RemoteAddr().String()}, []string{"red", "white", "magenta"})

	})

	// START OF MAIN ===============================================================

	// Command line arguments
	flag.BoolVar(&silentFlag, "s", false, "silent mode (default FALSE)")
	flag.StringVar(&keyFlag, "k", "/etc/ssh/ssh_host_ed25519_key", "server private key")
	flag.BoolVar(&loginFlag, "l", false, "allow clients to login and spawn a shell (default FALSE)")
	flag.IntVar(&portFlag, "p", 22, "port")
	flag.BoolVar(&verboseFlag, "v", false, "log to stdout NOT RECOMMENDED (default FALSE)")
	flag.StringVar(&messageFlag, "m", "", "send message to client on exit (default NONE)")
	flag.Parse()

	// use server's private key
	hostKeyFile := ssh.HostKeyFile(keyFlag)

	// enable password login
	passwordAuth := ssh.PasswordAuth(func(ctx ssh.Context, pass string) bool {

		printLog([]string{"LOGIN ATTEMPT\t\t", "Address:", ctx.RemoteAddr().String(), "  Client:", ctx.ClientVersion(), "  Username:", ctx.User(), "  Password:", pass}, []string{"yellow", "white", "magenta", "white", "white", "white", "magenta", "white", "magenta"})

		if loginFlag {
			// check if password is valid
			hash := findHash(ctx.User())
			// case where username is not in /etc/shadow
			if hash == "" {
				return false
			} else {
				new_hash := C.GoString(C.crypt(C.CString(pass), C.CString(hash)))
				return hash == new_hash
			}
		} else {
			return false
		}
	})

	printLog([]string{"STARTING SSHLOG"}, []string{"green"})
	printLog([]string{"CREATED LOG FILE\t", "Filename: .ServerLog"}, []string{"green", "white"})

	// start server
	startErr := ssh.ListenAndServe(":"+strconv.Itoa(portFlag), nil, hostKeyFile, passwordAuth)
	if startErr != nil {
		printLog([]string{"FAILED TO START SSHLOG\t", startErr.Error()}, []string{"red", "white"})
	}

}
