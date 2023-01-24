# SSHlog
SSH server that captures login credentials and logs client actions.  
A private key can be specified at runtime so this program appears to be the true SSH sever.  
Separate log files are created for every client in the directory the program is run.  

## Build
```bash
git clone https://github.com/Zilog-Z80/SSHlog.git
cd SSHlog
go build SSHlog.go
```

## Change SSH Server Port
```bash
nano /etc/ssh/sshd_config
# Uncomment the '#Port 22' line and change to desired port
systemctl reload ssh  
# This change may not survive reboot if the SSH server is socket activated
```

## SSHlog Usage
```bash
./SSHlog # Run with default settings
./SSHlog -h # display usage

Usage of ./SSHlog:
  -k string
    	server private key (default "/etc/ssh/ssh_host_ed25519_key")
  -l	prevent client login
  -m string
    	send message to client on exit
  -p int
    	port (default 22)
  -s	silent mode
  -v	log to stdout (NOT RECOMMENDED)
```