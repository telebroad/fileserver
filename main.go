package main

import (
	"fmt"
	"github.com/telebroad/ftpserver/server"
	"github.com/telebroad/ftpserver/users"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// this is the bublic ip of the server FOR PASV mode
	ftpServerIPv4 := os.Getenv("FTP_SERVER_IPV4")
	if ftpServerIPv4 == "" {

		// Set a default FTP_SERVER_IPV4 if the environment variable is not set
		fmt.Println("FTP_SERVER_IPV4 was empty so Getting public ip from ipify.org...")
		ipifyRes, err := http.Get("https://api.ipify.org")
		if err != nil {
			fmt.Println("Error getting public ip", "error", err)
			return
		}
		ftpServerIPv4b, err := io.ReadAll(ipifyRes.Body)
		if err != nil {
			fmt.Println("Error reading public ip", "error", err)
			return
		}
		ftpServerIPv4 = string(ftpServerIPv4b)
		fmt.Println("FTP_SERVER_IPV4 is ", ftpServerIPv4)
		// Set a default port if the environment variable is not set
	}
	ftpPort := os.Getenv("FTP_SERVER_PORT")
	if ftpPort == "" {
		// Set a default port if the environment variable is not set
		ftpPort = ":21"
	}
	ftpServerRoot := os.Getenv("FTP_SERVER_ROOT")
	if ftpServerRoot == "" {
		// Set a default port if the environment variable is not set
		ftpServerRoot = "/static"
	}
	Users := users.NewLocalUsers()
	user1 := Users.Add("user", "password", 1)
	user1.AddIP("127.0.0.1")
	user1.AddIP("::1")

	ftpServer, err := server.NewFTPServer(ftpPort, ftpServerIPv4, server.NewFtpLocalFS(ftpServerRoot, "/"), Users)
	if err != nil {
		fmt.Println("Error starting ftp server", "error", err)
		return
	}
	ftpServer.Start()
	stopChan := make(chan os.Signal, 1)
	signal.Notify(
		stopChan,
		syscall.SIGHUP,  // (0x1) Terminal hangup
		syscall.SIGINT,  // (0x2) Interrupt from keyboard (Ctrl+C)
		syscall.SIGQUIT, // (0x3) Quit from keyboard
		syscall.SIGABRT, // (0x6) Aborted (core dumped)
		syscall.SIGKILL, // (0x9) Killed (cannot be caught)
		syscall.SIGTERM, // (0xf) Terminated (generic termination signal)
	)

	<-stopChan
	ftpServer.Stop()
}
