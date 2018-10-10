#!/usr/bin/bash

set -xe

/usr/sbin/collectd -C /etc/collectd/collectd.conf
/usr/sbin/procwatch -U /var/run/collectd.sock /etc/procwatch.json
