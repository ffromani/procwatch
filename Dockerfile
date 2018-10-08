FROM fedora:28

MAINTAINER "Francesco Romani" <fromani@redhat.com>
ENV container docker

RUN \
  dnf install -y \
    collectd collectd-virt collectd-write_prometheus && \
  dnf clean all

RUN \
  dnf install -y \
    procps-ng curl less && \
  dnf clean all

COPY docker/collectd.conf /etc/collectd.conf
COPY cluster/procwatch.json /etc/procwatch.json
COPY procwatch /usr/local/libexec/procwatch

ENTRYPOINT ["/usr/sbin/collectd", "-f"]
