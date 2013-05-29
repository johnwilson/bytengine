#!/bin/sh

### BEGIN INIT INFO
# Provides:          bytengine-server
# Required-Start:    $all
# Required-Stop:     $all
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: bytengine-server content storage server
# Description:       bytengine-server content storage server
### END INIT INFO

PATH=/opt/bytengine-server/bin:/sbin:/bin:/usr/sbin:/usr/bin
DAEMON=/opt/bytengine-server/bin/bytengine
DAEMON_ARGS='--config /opt/bytengine-server/conf/config.json'
NAME=bytengine-server
DESC=bytengine-server

test -x $DAEMON || exit 0

set -e

case "$1" in
  start)
        echo -n "Starting $DESC: "

        start-stop-daemon --start --user bytengine -b -c bytengine:bytengine \
            --startas $DAEMON -- $DAEMON_ARGS

        echo "$NAME."
        ;;
  stop)
        echo -n "Stopping $DESC: "

        start-stop-daemon --stop --exec $DAEMON -c bytengine:bytengine \
            $DAEMON -- $DAEMON_ARGS

        echo "$NAME."
        ;;
      *)    
            N=/etc/init.d/$NAME
            echo "Usage: $N {start|stop}" >&2
            exit 1
            ;;
    esac
    
    exit 0