#!/bin/bash

DIR=$(dirname $0)

export ETCD=$(curl https://discovery.etcd.io/new)

echo generating user-data with $ETCD
echo URI saved to \$ETCD

#sed -i "s|^                discovery:.*$|                discovery: ${ETCD}|" deis_rax.yaml
#sed -i "s|^                discovery:.*$|                discovery: ${ETCD}|" deis_rax_onmetal.yaml