VMI metrics reporting for kubevirt
==================================

Introduction
------------

We integrate in the base flow that [kubevirt recently built](https://github.com/kubevirt/kubevirt/pull/1515), but for the moment
we want to coexist side-by-side: VMI metrics will be available through a separate ServiceMonitor.

Installation
------------

0. create the collectd config map using `../collectd-config-map.yaml`
1. create the service which will be used by the later ServiceMonitor using `collectd-svc.yaml`
2. deploy collectd in your cluster using `../collectd-node-agent-$PLATFORM.yaml` where PLATFORM is either "ocp" or "k8s". Example: `collectd-node-agent-ocp.yaml`
3. now you can create the ServiceMonitor using `collectd-svc-monitor.yaml`


Deploying prometheus server in your cluster
-------------------------------------------

Use the instructions in the [prometheus operator user guide](https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/getting-started.md)
there is an [example in the kubevirt documentation too](https://github.com/kubevirt/kubevirt/pull/1515)

Caveat/gotchas:
0. configure rbac for prometheus server too (not just for the operator). This is somehow out of order in the documentation and may be slightly misleading.

