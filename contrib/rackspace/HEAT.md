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
$ export DEIS_NUM_INSTANCES=3
$ export STACK=deis
$ heat stack-create $STACK --template-file ./deis_rax.yaml \
         -P flavor='2 GB Performance' -P count=$DEIS_NUM_INSTANCES \
         -P name="$STACK"
```


#### Or Create a 3 node OnMetal cluster via Heat:

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

### Initialize the cluster

#### From local machine

Save your SSH private key from heat output.

```
DEIS_KEY=$(heat stack-show $STACK | grep RSA | awk -F\" '{print $4}') && printf $DEIS_KEY > ~/deis_key && chmod 0600 ~/deis_key
ssh-add ~/deis_key
```

If you have `fleetctl` installed locally, get the public and private IPs of one of your nodes, and issue a `make run` from the project root:
```console
$ export FLEETCTL_TUNNEL=23.253.219.94
$ export FLEETCTL_ENDPOINT=http://10.21.12.1:4001
export DEIS_NUM_ROUTERS=$DEIS_NUM_INSTANCES
$ cd ../.. && make run
```

The script will deploy Deis and make sure the services start properly.

### Configure Load Balancer
Heat created a load balander for us, but we need to create a second one from the WebUI for SSH.  Hopefully we can do this via heat as well, but I haven't worked out how yet.

    Load Balancer 2
    Virtual IP Shared VIP on Another Load Balancer (select Load Balancer 1)
    Port 2222
    Protocol TCP

### Configure DNS
You'll need to configure DNS records so you can access applications hosted on Deis. See [Configuring DNS](http://docs.deis.io/en/latest/installing_deis/configure-dns/) for details.


### Use Deis!
After that, register with Deis!
```console
$ deis register http://deis.example.org
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
