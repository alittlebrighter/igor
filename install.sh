#!/usr/bin/env bash

echo "setting up a new Igor brain"

cd systemd
SVCS=`ls -a *.service`
for svc in $SVCS ; do
    systemctl stop $svc
done

# download binaries
cd ..
echo "downloading binaries..."
cp assets/* /usr/bin/

mkdir -p /opt/igor/data

# setup systemd services
echo "configuring systemd services..."
cd systemd
cp *.service /etc/systemd/system/

systemctl daemon-reload

echo "starting services..."
for svc in $SVCS ; do
    systemctl stop $svc
    systemctl enable $svc
    systemctl start $svc
done

echo "finished, this brain is now ALIVE!"
cd $PWD