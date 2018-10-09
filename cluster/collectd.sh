#!/usr/bin/bash

set -xe

/usr/sbin/collectd
/usr/sbin/procwatch -U /var/run/collectd.sock /etc/procwatch.json
