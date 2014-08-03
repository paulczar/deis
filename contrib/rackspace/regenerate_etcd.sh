#!/bin/bash

DIR=$(dirname $0)

ETCD=$(curl https://discovery.etcd.io/new)

echo generating user-data with $ETCD

sed -i "s|^                discovery:.*$|                discovery: ${ETCD}|" deis_rax.yaml
sed -i "s|^                discovery:.*$|                discovery: ${ETCD}|" deis_rax_onmetal.yaml