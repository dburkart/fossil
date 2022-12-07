# Deployment

- [Deployment](#deployment)
  - [Containerized](#containerized)
    - [Docker](#docker)
    - [Kubernetes](#kubernetes)
      - [Deploy](#deploy)
      - [Config](#config)
  - [Binary Distributions](#binary-distributions)
    - [FreeBSD](#freebsd)


Fossil recommendations
- Memory
  - Usage: 500Mi
- CPU
  - Usage: 500m



## Containerized

### Docker

Build
```shell
docker build -t fossil .
```
or Pull
```shell
docker pull gideonw/fossil:latest
```
and Run
```shell
docker run -it -p8001:8001 -p2112:2112 gideonw/fossil:latest server
```

### Kubernetes
Apply the following resources from the `deploy/kubernetes` directory within the repo.

Objects:
- Deployment
- Service
- Ingress

#### Deploy
Pull and clone the repository in-order to use the deploy/kubernetes example deployment.
```shell
kubectl create namespace fossil
```
```shell
kubectl config set-context --current --namespace=fossil
```
```shell
kubectl apply -f ./deploy/kubernetes/example
```

#### Config
Modify the `deploy/kubernetes/example/config.yaml` file to update the configuration for the fossil deployment.

## Binary Distributions

We don't provide any binary distributions as of this writing, but you can install the fossil command 
in the typical way you would install a go package:

```shell
go install github.com/dburkart/fossil
```

Depending on how and where you are deploying fossil, you may want run the above command as a separate
fossil-specific user.

### FreeBSD
Below is an example FreeBSD daemon that you can install under `/usr/local/etc/rc.d/fossil`:

```shell
#!/bin/sh

# PROVIDE: fossil
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name=fossil
rcvar=fossil_enable

load_rc_config $name

: ${fossil_enable="NO"}
: ${fossil_home_dir:="<HOME>"}

pidfile="/var/run/${name}.pid"
procname="${fossil_home_dir}/go/bin/fossil"
command="/usr/sbin/daemon"
command_args="-S -p ${pidfile} -u <USER> ${procname} server --config /usr/local/etc/fossil/config.toml"

run_rc_command "$1"
```

Replace `<HOME>` with the location of user you are running the daemon under's home, and `<USER>` with
the user you'd like to run the fossil server under. You can also find the above script in "./deploy/freebsd/fossil-daemon"

Additionally, this script directs fossil to read `/usr/local/etc/fossil/config.toml`, but you can choose
to rely on the fossil autoload search path.

Now you can run the fossil database in the typical FreeBSD way, via `service fossil start`.
