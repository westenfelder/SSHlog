# SSHlog
![example](example.png)

## Build
```bash
git clone https://github.com/Zilog-Z80/SSHlog.git
cd SSHlog
go build SSHlog.go
./SSHlog
```

## Kill True SSH Server
`killall sshd` - kill all ssh processes  

## Change SSH Server Port
Ubuntu 20:  
```bash
nano /etc/ssh/sshd_config
# Uncomment the '#Port 22' line and change to desired port
systemctl restart ssh  
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