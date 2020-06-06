#!/usr/bin/env bash

echo "setting up a new Igor brain"

cd systemd
SVCS=`ls -a *.service`
for svc in $SVCS ; do
    systemctl stop $svc
done

# download/build binaries
cd ..
# TO DO
echo "downloading binaries..."
# for now, just creating a dist directory with all of the necessary tools built for your architecture will work
cp dist/* /usr/bin/

chmod +x /scripts/*.sh
cp scripts/* /usr/bin/

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