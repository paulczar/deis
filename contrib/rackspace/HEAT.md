# Provision a Deis Cluster on Rackspace

We'll mostly be following the [CoreOS on Rackspace](https://coreos.com/docs/running-coreos/cloud-providers/rackspace/) guide. You'll need to have a sane python environment with pip already installed (`sudo easy_install pip`) as well as a golang build environment. 

We have taken precautions to try and ensure a secure installation ( example: etcd not listening on external interface) and have tried to do as much
of the heavy lifting in the heat templates, so that there is few manual steps.

### Install fleetctl:
```
$ git clone https://github.com/coreos/fleet.git
$ cd fleet
$ ./build
$ sudo mv bin/fleetctl /usr/local/bin/
```

### Install Heat and its dependencies:
```console
$ sudo pip install python-heatclient 
$ sudo pip install python-novaclient
```

### Configure Openstack Authentication

_if you already have these set up you can skip this step._

Edit `~/.openrc` to match the following.
```
export OS_AUTH_URL=https://identity.api.rackspacecloud.com/v2.0/
export OS_USERNAME={rackspace_username}
export OS_PASSWORD={rackspace_api_key}
export OS_TENANT_ID={rackspace_account_number}
export OS_REGION_NAME=IAD
export HEAT_URL=https://iad.orchestration.api.rackspacecloud.com/v1/${OS_TENANT_ID}
```

Your account number is displayed in the menu in the upper right-hand corner of the cloud control panel UI, and your API key can be found on the Account Settings page.   If you want to use OnMetal Region must be set to IAD.

you then need to source the authentication settings.

```console
$ source ~/.openrc
```


### Create userdata with new etcd discovery URL

This needs to be run _prior_ to any time that you create a new heat stack:

```console
./regenerate_etcd.sh
```

### Launch the stack

#### Create a 3 node VM cluster via Heat:

```console
$ export DEIS_NUM_INSTANCES=3 && \
  export STACK=deis && \
  export ETCD=$(curl https://discovery.etcd.io/new) && \
  heat stack-create $STACK --template-file ./deis_rax.yaml \
         -P flavor='2 GB Performance' -P count=$DEIS_NUM_INSTANCES \
         -P name="$STACK" -P etcd_discovery="$ETCD"
```


#### Or Create a 3 node OnMetal cluster via Heat:

_experimental, not fully featured yet_

```console
$ export DEIS_NUM_INSTANCES=3
$ export STACK=deis_onmetal
$ heat stack-create $STACK --template-file ./deis_rax_onmetal.yaml \
          -P flavor='OnMetal Compute v1' -P count=$DEIS_NUM_INSTANCES \
         -P name="$STACK"
```

Note that for scheduling to work properly, clusters must consist of at least 3 nodes and always have an odd number of members.
For more information, see [optimal etcd cluster size](https://github.com/coreos/etcd/blob/master/Documentation/optimal-cluster-size.md).

Deis clusters of less than 3 nodes are unsupported.

### Watch your heat stack until it's created

the `stack_status` should read `CREATE_COMPLETE`

#### Linux

This will check the status of your stack every 2 seconds.

```console
$ watch heat stack-list
```

#### OSX

Run this every few minutes,  or chuck it in a loop.

```console
$ heat stack-list
```


### Prepare environment for fleet/etcd clients

Save your SSH private key from heat output.

```console
$ DEIS_KEY=$(heat stack-show $STACK | grep RSA | awk -F\" '{print $4}') && printf $DEIS_KEY > ~/deis_key && chmod 0600 ~/deis_key
$ ssh-add ~/deis_key
```


#### From local machine

If you have `fleetctl` installed locally, get the public and private IPs of one of your nodes:
```console
$ export FLEETCTL_TUNNEL=23.253.219.94
$ export FLEETCTL_ENDPOINT=http://10.21.12.1:4001
```

#### From coreOS server

get the public and pritate IPs of your one of your nodes:

```console
$ scp ~/deis_key core@23.253.219.94:/tmp/key
$ ssh core@23.253.219.94
$ chmod 0600 /tmp/key && eval `ssh-agent` && ssh-add /tmp/key
$ export FLEETCTL_ENDPOINT=http://10.21.12.1:4001
```

#### Check that deis install worked correctly

```console
$ etcdctl get /deis_install/complete
true
$ etcdctl get /deis_install/installed_by
10.21.12.2:4001
$ fleetctl list-units
UNIT        STATE   LOAD  ACTIVE  SUB DESC  MACHINE
deis-builder-data.service loaded    loaded  active  exited  - e01cc352.../162.242.218.233
deis-builder.service    launched  loaded  active  running - e01cc352.../162.242.218.233
deis-cache.service    launched  loaded  active  running - e01cc352.../162.242.218.233
deis-controller.service   launched  loaded  active  running - e01cc352.../162.242.218.233
deis-database-data.service  loaded    loaded  active  exited  - e01cc352.../162.242.218.233
deis-database.service   launched  loaded  active  running - e01cc352.../162.242.218.233
deis-logger-data.service  loaded    loaded  active  exited  - e01cc352.../162.242.218.233
deis-logger.service   launched  loaded  active  running - e01cc352.../162.242.218.233
deis-registry-data.service  loaded    loaded  active  exited  - e01cc352.../162.242.218.233
deis-registry.service   launched  loaded  active  running - e01cc352.../162.242.218.233
deis-router@1.service   launched  loaded  active  running - e01cc352.../162.242.218.233
```

if deis failed to install ( the etcd keys are empty ) then you can rerun the install from any of the nodes

```console
$ systemctl start install_deis
```

or you can run it manually by pulling the docker command out of the service.

```console
$ grep ExecStart= /etc/systemd/system/install_deis.service | awk -F\' '{print $2}' | sh
```

The script will deploy Deis and make sure the services start properly.

### Configure Load Balancer
Heat created a load balander for us, but we need to create a second one that shares the same VIP from the Rackspace Cloud UI for SSH.  Hopefully we can do this via heat as well, but I haven't worked out how yet.

    Load Balancer 2
    Virtual IP Shared VIP on Another Load Balancer (select Load Balancer 1)
    Port 2222
    Protocol TCP

### Configure DNS
You'll need to configure DNS records so you can access applications hosted on Deis. See [Configuring DNS](http://docs.deis.io/en/latest/installing_deis/configure-dns/) for details.

If you don't have a domain to use, or you're just testing you can use the IP of your LB in a xip.io URI ( example:  50.56.167.203.xip.io )

### Use Deis!
After that, register with Deis!
```console
$ deis register http://deis.50.56.167.203.xip.io
username: deis
password:
password (confirm):
email: info@opdemand.com
```

## Hack on Deis
If you'd like to use this deployment to build Deis, you'll need to set `DEIS_HOSTS` to an array of your cluster hosts:
```console
$ DEIS_HOSTS="1.2.3.4 2.3.4.5 3.4.5.6" make build
```

This variable is used in the `make build` command.
