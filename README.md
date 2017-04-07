procwatch
=========

Watch processes and report their usage consumption (CPU, memory) to
collectd.

License
=======

MIT

Dependencies
============

[gopsutil](https://github.com/shirou/gopsutil)

Installation
============

0. Make sure you have the golang toolset installed on your box. For example, on
   Fedora:

   ```
   # dnf install golang-bin
   ```

  If this is your first golang application, make sure you have the GOPATH set:

  ```
  $ export GOPATH="$HOME/go"
  ```

You may want to make this setting persistent

1. checkout the sources, and transparently build the tool

  ```
  $ go get github.com/fromanirh/procwatch
  ```
  
2. copy the tool on your filesystem:

  ```
  $ sudo cp $GOPATH/bin/procwatch /usr/local/libexec
  ```

3. fix the SELinux configuration:

  ```
  # semanage fcontext -a -t collectd_exec_t /usr/local/libexec/procwatch
  # restorecon -v /usr/local/libexec/procwatch
  ```

4. copy the recommended configuration:

  ```
  $ sudo cp $GOPATH/src/github.com/fromanirh/procwatch/conf/procwatch.json /etc
  ```

5. copy the collectd configlet:

  ```
  $ sudo cp $GOPATH/src/github.com/fromanirh/procwatch/conf/collectd/procwatchvdsm.conf /etc/collectd.d
  ```

6. restart collectd

  ```
  # systemctl restart collectd
  ```

7. Done!

   ``` 
   # collectdctl listval | grep exec
   kenji.rokugan.lan/exec-vdsmd-4615/cpu-perc
   kenji.rokugan.lan/exec-vdsmd-4615/cpu-system
   kenji.rokugan.lan/exec-vdsmd-4615/cpu-user
   kenji.rokugan.lan/exec-vdsmd-4615/memory-resident
   kenji.rokugan.lan/exec-vdsmd-4615/memory-virtual
   kenji.rokugan.lan/exec-vdsmd-4615/percent-cpu
   ```

How it works
============

TODO


TODO
====

* test suite

* track other processes (MOM, libvirtd)

* support multiple tool (netdata?)

