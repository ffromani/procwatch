procwatch
=========

Watch processes and report their usage consumption (CPU, memory).

License
=======

Apache v2

Dependencies
============

* [gopsutil](https://github.com/shirou/gopsutil)
* [kubernetes APIs](https://github.com/kubernetes/kubernetes)


Installation: kubernetes/kubevirt cluster
=========================================

This project can be deployed in a kubevirt cluster to report metrics about processes running inside PODs.
You may use this to monitor the resource consumption of these infrastructure processes for VM-based workloads.
We assume that the cluster is running [prometheus-operator](https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/getting-started.md) to manage the monitoring using prometheus,
and [kubevirt](https://github.com/kubevirt/kubevirt/releases/tag/v0.9.1) >= 0.9.1, which includes itself better integration with prometheus operator.

0. First deploy a new service to plug in the configuration KubeVirt uses to interact with prometheus-operator:
```
kubectl create -f procwatch/cluster/kube-service-vmi.yaml
```

1. Now deploy the tool itself into the cluster:
Set PLATFORM to either "k8s" or "ocp" and
```
kubectl create -f procwatch/cluster/collectd-config-map.yaml
kubectl create -f procwatch/cluster/collectd-node-agent-$PLATFORM.yaml
```

2. procwatch installs a new deployment in the `kube-system` namespace. VM pods usually run in the `default` namespace.
This may make the prometheus server unable to scrape the metrics endpoint.
procwatch added. To fix this, deploy your prometheus server in your cluster like this:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
spec:
  serviceAccountName: prometheus
  serviceMonitorNamespaceSelector:
    matchLabels:
      prometheus.kubevirt.io: ""
  serviceMonitorSelector:
    matchLabels:
      prometheus.kubevirt.io: ""
  resources:
    requests:
      memory: 400Mi

```
Note the usage of `serviceMonitorNamespaceSelector`. [See here for more details](https://github.com/coreos/prometheus-operator/issues/1331)

3. Now, you may need to add labels to the namespaces, like kube-system. Here's an example of how it could look like:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  ...
  creationTimestamp: 2018-09-21T13:53:16Z
  labels:
    prometheus.kubevirt.io: ""
...
```

3. The last step: now you need to deploy a `Service Monitor` to let `prometheus-operator` pickup and add rules for this endpoint:
```
kubectl create -f procwatch/cluster/kube-service-monitor-vmi.yaml
```

Please check the next sections for Caveats.


Notes about integration with kubernetes/kubevirt
================================================

**WARNING**
In order to resolve the PIDs to meaningful VM domain names, procwatch **needs to access the CRI socket on the host**. The default YAMLs do that exposing the host /var/run, because by default
the CRI socket sits in /var/run (and not in a subdirectory, which would make things easier).
**YOU MAY WANT TO REVIEW THIS SETTING BEFORE TO DEPLOY PROCWATCH ON YOUR CLUSTER**

If you disable the CRI socket access, procwatch will just report the PIDs of the monitored processes.


Installation: bare metal
========================

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

0. (TODO): ensure the vendored dependencies

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

4. copy the recommended configurations:

  ```
  # mkdir /etc/procwatch.d
  $ sudo cp $GOPATH/src/github.com/fromanirh/procwatch/conf/procwatch/*.json /etc/procwatch.d/
  ```

5. copy the collectd configlets:

  ```
  $ sudo cp $GOPATH/src/github.com/fromanirh/procwatch/conf/collectd/procwatch*.conf /etc/collectd.d/
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

