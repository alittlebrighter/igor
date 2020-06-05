#!/usr/bin/env bash

lanRoutes=`/usr/bin/lansrv -scan -service nats-cluster`
echo "found routes to cluster: $lanRoutes"

/usr/bin/nats-server -a 127.0.0.1 --cluster nats://0.0.0.0:4248 --routes "$lanRoutes"