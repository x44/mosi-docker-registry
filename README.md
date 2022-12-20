[![CI](https://github.com/x44/mosi-docker-registry/actions/workflows/ci.yml/badge.svg)](https://github.com/x44/mosi-docker-registry/actions/workflows/ci.yml)
[![Release](https://github.com/x44/mosi-docker-registry/actions/workflows/release.yml/badge.svg)](https://github.com/x44/mosi-docker-registry/actions/workflows/release.yml)

# Mosi Docker Registry
The horrible sounding **Mosi** stands for **Most Simple**<br>

Mosi is a minimalistic Docker registry with a very small memory footprint, has a simple user account management and can - but does not need to - be installed as a system service.<br>
The system service functionality is powered by https://github.com/kardianos/service

**Please note that Mosi comes without any warranty!**

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
  - [Default Config](#default-config)
  - [Config File](#config-file)
  - [TLS Mode Config](#tls-mode-configuration)
  - [Non-TLS Mode Config](#non-tls-mode-configuration)
- [Running](#running-mosi)

## Installation
- [Download the latest release](https://github.com/x44/mosi-docker-registry/releases/latest)
- Extract the downloaded zip file to a directory of your choice

## Configuration
The configuration is stored in the file `config.json` in the `conf` sub directory.

### Default Config
If you are using Mosi for the first time, run it without any arguments to create a default config file and a self-signed server certificate:
```
mosi
```
### Config File
```json
{
	"server": {
		"host": "mosi",
		"port": 443,
		"bind": "",
		"tlsCrtFile": "certs/mosi-default.crt",
		"tlsKeyFile": "certs/mosi-default.key"
	},
	"proxy": {
		"host": "",
		"port": 0
	},
	"log": {
		"serviceLevel": "INFO",
		"consoleLevel": "INFO",
		"logFileLevel": "INFO"
	},
	"repo": {
		"dir": "repo",
		"allowAnonymousPull": true
	},
	"accounts": [
		{
			"usr": "admin",
			"pwd": "admin",
			"admin": true,
			"images": [
				{
					"name": "*",
					"pull": true,
					"push": true
				}
			]
		},
		{
			"usr": "anonymous",
			"pwd": "",
			"admin": false,
			"images": [
				{
					"name": "*",
					"pull": true,
					"push": false
				}
			]
		}
	]
}
```
| Section  | Key                 | Description |
|----------|---------------------|-------------|
| server   | host                | Server host name. Must match the DNS name of the server certificate, when using TLS.|
| server   | port                | Port to bind the server to. |
| server   | bind                | Optional IP address or host name to bind the server to. |
| server   | tlsCrtFile          | Relative or absolute path of the server certificate file. Required in TLS mode. Leave empty for non-TLS mode. |
| server   | tlsKeyFile          | Relative or absolute path of the server certificate key file. Required in TLS mode. Leave empty for non-TLS mode.  |
| proxy    | host                | Proxy host name. Required if the server is running behind a TLS terminating reverse proxy. |
| proxy    | port                | Proxy port. Required if the server is running behind a TLS terminating reverse proxy. |
| log      | serviceLevel        | Syslog level. Supported levels are `DEBUG` `INFO` `WARN` `ERROR` `SILENT`|
| log      | consoleLevel        | Console level.  |
| log      | logFileLevel        | Log file level. The log file is located in the `log` sub directory. |
| repo     | dir                 | Relative or absolute repository storage directory. |
| repo     | allowAnonymousPull  | Whether to allow pull requests by the `anonymous` user account. |
| accounts |                     | List of user accounts. |
| accounts | usr                 | Account user name. |
| accounts | pwd                 | Account password. |
| accounts | admin               | Whether the user account has admin rights. |
| accounts | images              | List of images the user account has access to. |
| images   | name                | Image name or pattern the user account has access to. |
| images   | pull                | Whether the user account may pull. |
| images   | push                | Whether the user account may push. |

## TLS Mode Configuration
Mosi starts in TLS mode if the config fields `server.tlsCrtFile` and `server.tlsKeyFile` are not empty.

To run Mosi in TLS mode...
- A valid TLS certificate and key file are required
- `server.name` must match one of the DNS entries in the TLS certificate file
- The server's IP address must match one of the IP entries in the TLS certificate file
- `server.tlsCrtFile` and `server.tlsKeyFile` must point to the TLS certificate and key file respectively

### Create a Self-Signed Certificate
You can either
- Use `tools/generate-server-certificate` from the Mosi binary distribution
- Use `scripts/generate-server-certificate.sh` from the Mosi binary distribution
- Or follow these steps:

1) Create a certificate request file 'mosi.cfg' with the following content.<br>
Replace DNS:mosi with the server name you configured in `server.name`<br>
Replace IP:127.0.0.1,IP:192.168.1.2 with your server's IP address(es)

```ini
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no
[req_distinguished_name]
O = Mosi Docker Registry
[v3_req]
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid,issuer
basicConstraints = CA:TRUE
keyUsage = keyCertSign, dataEncipherment, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:mosi,IP:127.0.0.1,IP:192.168.1.2
```

2) Run the following command to generate the .crt and .key file
```
openssl req -x509 -nodes -days 36500 -newkey rsa:2048 -keyout mosi.key -out mosi.crt -config mosi.cfg -extensions 'v3_req'
```

### Using a Self-Signed Certificate
To use Mosi with a self-signed certificate we need to make the client(s) accept our self-signed certificate.

#### Docker Desktop (Windows)
You can either 
- Add the self-signed certificate to the Windows Certificate Manager:
  - Right-click the mosi.crt file and choose **Install Certificate**
  - Choose **Current User** or **Local Machine**
  - Choose **Place all certificates in the following store** and click **Browse...**
  - Choose **Trusted Root Certificate Authorities**
  - Click **Next** / **Finish**
  - **Restart Docker Desktop**
     
OR
- Copy the self-signed certificate file to the Docker Config Directory:
	- Create the directory `%USERPROFILE%\.docker\certs.d`
	- In this directory create a directory for the Mosi hostname and port. The name of this directory must be in the format `host` or `host port` (with a **SPACE** between host and port) and it must match the server name and port which you are going to use. For example, `docker login mosi` requires the directory name to be just `mosi`. `docker login mosi:443` requires the directory name to be `mosi 443`
	- Copy the mosi.crt file into this directory. Note that you do **not** have to rename this file to ca.crt
    - Example for mosi:443
		`
		%USERPROFILE%\.docker\certs.d\mosi 443\mosi.crt
		`
	- **Restart Docker Desktop**


#### Docker Toolbox (Windows)
You can either
- Use `tools/configure-docker-toolbox` from the Mosi binary distribution
- Use `scripts/configure-docker-toolbox.sh` from the Mosi binary distribution
- Or follow the steps below:

Please note that we **must** use the directory %USERPROFILE% since this directory gets mounted as a shared folder in the Docker VM.

1) Copy the self-signed certificate to the %USERPROFILE% directory
```
copy mosi.crt %USERPROFILE%
```

2) Create %USERPROFILE%\bootlocal.sh with LF line endings:
```
#!/bin/bash
sleep 5
sudo -i
echo "192.168.1.2 mosi" >> /etc/hosts
exit
```

3) SSH into the Docker VM
```
docker-machine ssh default
```

4) Run the following commands
```
sudo -i
mkdir -p /var/lib/boot2docker/certs
cp /c/Users/YOURNAME/mosi.crt /var/lib/boot2docker/certs
cp /c/Users/YOURNAME/bootlocal.sh /var/lib/boot2docker
chmod +x /var/lib/boot2docker/bootlocal.sh
exit
exit
```

5) Back from the SSH session run
```
docker-machine restart
```

6) Add the registry IP address and host name to C:\Windows\System32\drivers\etc\hosts
```
192.168.1.2 mosi
```

#### Kubernetes
On all master and worker nodes...

1) Add the registry IP address and Hostname to /etc/hosts
```
sudo tee -a /etc/hosts <<EOF
192.168.1.2 mosi
EOF
```

2) Get the self-signed certificate from the registry server
```
openssl s_client -showcerts -connect mosi:4444 < /dev/null | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > ca.crt
```

3) Add the self-signed certificate
```
sudo mv ca.crt /etc/ssl/certs
sudo update-ca-certificates
```

#### Minikube

1) Copy the self-signed certificate
```
copy /Y mosi.crt %USERPROFILE%\.minikube\certs\
```

2) Start Minikube
```
minikube start --embed-certs
```

## Non-TLS Mode Configuration
Mosi starts in Non-TLS mode if either one of the config fields `server.tlsCrtFile` or `server.tlsKeyFile` is empty.

To run Mosi in Non-TLS mode...
- The reverse proxy's host and port must be configured in the `proxy` config section

### Reverse Proxy Configuration for Non-TLS Mode
Example nginx config. The nginx reverse proxy is at mosiproxy:443 and forwards to Mosi at 192.168.1.2:4444
```nginx
server {
	listen       443 ssl;
	server_name  mosiproxy;

	# Allow upload of large files
	client_max_body_size 20G;

	ssl_certificate      certs/mosiproxy.crt;
	ssl_certificate_key  certs/mosiproxy.key;

	location /v2/ {
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Port $server_port;
		proxy_pass http://192.168.1.2:4444;
	}
}
```

### Mosi Configuration for Non-TLS Mode
Example Mosi config. The reverse proxy is at mosiproxy:443 and forwards to Mosi at mosi:4444
```json
"server": {
	"host": "mosi",
	"port": 4444,
	"bind": "192.168.1.2",
	"tlsCrtFile": "",
	"tlsKeyFile": ""
},
"proxy": {
	"host": "mosiproxy",
	"port": 443
}
```

## Running Mosi
To start Mosi as a "normal" program run
```
mosi
```
For help run
```
mosi -h
```
To install / uninstall the system service run
```
mosi install
```
```
mosi uninstall
```
To start / restart / stop the system service run
```
mosi start
```
```
mosi restart
```
```
mosi stop
```
To get the status of the system service run
```
mosi status
```
You can also combine service commands, for example
```
mosi install start status
```

