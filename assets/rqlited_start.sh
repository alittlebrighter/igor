#!/usr/bin/env bash

NODE_ID=`head /dev/urandom | tr -dc A-Za-z0-9 | head -c 13 ; echo ''`
CLUSTER=`/usr/bin/lansrv -scan -service rqlited`

/usr/bin/rqlited -node-id "igor-$NODE_ID" -on-disk -raft-addr 0.0.0.0:4002 -join "$CLUSTER" /opt/igor/data