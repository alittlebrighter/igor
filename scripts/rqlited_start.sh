#!/usr/bin/env bash

lanRoutes=`/usr/bin/lansrv -scan -service rqlited`
echo "found routes to cluster: $lanRoutes"

nodeId=`head /dev/urandom | tr -dc A-Za-z0-9 | head -c 13 ; echo ''`

/usr/bin/rqlited -node-id "igor-$nodeId" -on-disk -raft-addr 0.0.0.0:4002 -join "$lanRoutes" /opt/igor/data