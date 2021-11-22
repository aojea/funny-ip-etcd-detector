# Funny IPs detector for Kubernetes clusters

Since golang 1.17, [IPv4 addresses with leading zeros are rejected by the standard library](https://github.com/golang/go/issues/30999).

The rationale behind this decision is perfectly explained by Russ cox @rsc in this comment:

> We are treating the change as a robustness improvement and not a security fix due to its potential for breaking working use cases.
>
> The situation is not nearly so clear cut as the advocates of CVE-2021-29923 would have people believe. They present it as a bug, plain and simple, not to treat leading zeros in IP addresses as indicating octal numbers, but that's not obvious. The BSD TCP/IP stack introduced the octal parsing, perhaps even accidentally, and because BSD is the most commonly used code, that interpretation is also the most common one. But it's not the only interpretation. In fact, the earliest IP RFCs directly contradict the BSD implementation - they are pretty clear that IP addresses with leading zeros are meant to be interpreted as decimal.
>
> Furthermore, the claimed vulnerability is like a TOCTTOU problem where the check and use are handled by different software with differing interpretation of leading zeros. The right fix is, as it always is, to put the check and use together.
>
> Rejecting the leading zeros entirely avoids resolving the radix ambiguity the wrong way, which improves robustness. But it can also break existing code that might be processing config files that contain leading zeros and were happy with the radix-10 interpretation.
>
> Given that
>
> 1. the right fix for any security consequence does not involve this change at all (the right fix is to place the check and use in the same program), and that
> 2. the Go behavior is entirely valid according to some RFCs, and that
> 3. the change has a very real possibility of breaking existing, valid, working use cases,
>
> we chose to make the change only in a new major version, as a robustness fix, rather than treat it as an urgent security fix that would require backporting. We do not want to push a breaking change that will keep people from being able to pick up critical Go 1.16 patches later.

## What are funny IPs

The adjective is because of the post [curl those funny IPv4 addresses](https://daniel.haxx.se/blog/2021/04/19/curl-those-funny-ipv4-addresses/))
that explains the history and the problem of liberal parsing of IP addresses and the consequences and security
risks caused the lack of normalization, mainly due to the use of different notations to abuse parsers misalignment
to bypass filters.

## IPv6

Surprisingly, IPv6 has not problem :)

## Kubernetes

Kubernetes uses golang, the change has been tracked in [Guard against stdlib ParseIP/ParseCIDR changes in API validation](https://github.com/kubernetes/kubernetes/issues/100895)
Jordan Liggit @liggit explains the impact of the golang change in Kubernetes in the following comment:

> While generally positive for new data, this could have the effect of making existing persisted data suddenly be considered invalid. That would cause objects to fail validation on updates that did not modify the existing suddenly-invalid fields, and could prevent objects with that data from being able to be successfully deleted (for example, an object with a finalizer that has to be removed via update prior to deletion, and an immutable field validated with ParseIP/ParseCIDR)

In order to solve of the problem, Kubernetes has decided to keep the loose validation on IP addresses, [using the loose parser for IPv4 addresses](https://github.com/kubernetes/kubernetes/pull/104368)

But this still exposes users that doesn't validate the IPv4 addresses with leading zeros.

## funny-ip-etcd-detector

Kubernetes stores it's data in etcd, `funny-ip-etcd-detector` is a tool that is able to parse an etcd database
and return the etcd keys with IPv4 addresses with leading zeros, so the user can identify and normalize them.

```sh
funny-ip-etcd-detector inspects etcd db files for finding IPv4 addresses with leading zeros.

Usage:
  funny-ip-etcd-detector [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  find-ips    find-ips lists IPv4 addressess with leading zeroes that will be rejected since golang 1.17 (ref: golang/go#30999).
  help        Help about any command

Flags:
  -h, --help               help for funny-ip-etcd-detector
      --timeout duration   time to wait to obtain a file lock on db file, 0 to block indefinitely (default 10s)

Use "funny-ip-etcd-detector [command] --help" for more information about a command.
```

## Installation

You can install the `funny-ip-detector` using `go install github.com/aojea/funny-ip-etcd-detector@latest` or
checking out the repository and compiling it locally:

```sh
git clone https://github.com/aojea/funny-ip-etcd-detector.git
cd funny-ip-etcd-detector
make
```

The binary will be generated in the `./bin` folder.


## How to use it

[Snapshot the etcd keyspace](https://kubernetes.io/docs/tasks/administer-cluster/configure-upgrade-etcd/#built-in-snapshot)

A snapshot may either be taken from a live member with the etcdctl snapshot save command or by copying the member/snap/db file from an etcd data directory that is not currently used by an etcd process.

```sh
$ ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=<trusted-ca-file> --cert=<cert-file> --key=<key-file> \
  snapshot save <backup-file-location>
```

Use the `funny-ip-detector` on the obtained snapshot, it will exit with an error if IPv4 address with leading zeros are found:

```
$ funny-ip-etcd-detector find-ips /var/lib/etcd/member/snap/db
WARNING Invalid IPv4 addresses ["1.01.1.2"] on key: "/registry/services/specs/default/nginx-deployment"
2021/11/22 16:29:50 Invalid IPv4 addresses found
```

You can use the tool to dump all IPv4 addresses in case you want to audit all the IPv4 addresses stored:

```sh
$ funny-ip-etcd-detector find-ips --match-all snapshot.db
...
IPv4 addresses found ["10.244.0.0"] on key: "/registry/controllerrevisions/kube-system/kindnet-6445d85c5c"
IPv4 addresses found ["172.18.0.2" "127.0.0.1" "10.244.0.0" "10.96.0.0" "127.0.0.1" "127.0.0.1" "172.18.0.22" "172.18.0.2" "172.18.0.2"] on key: "/registry/pods/kube-system/kube-controller-manager-kind-control-plane"
IPv4 addresses found ["172.18.0.2" "172.18.0.2" "172.18.0.2" "172.18.0.2" "172.18.0.2" "127.0.0.1" "172.18.0.2" "127.0.0.1" "172.18.0.2" "127.0.0.1" "127.0.0.1" "172.18.0.22" "172.18.0.2" "172.18.0.2"] on key: "/registry/pods/kube-system/etcd-kind-control-plane"
IPv4 addresses found ["172.18.0.2" "172.18.0.2" "172.18.0.2" "127.0.0.1" "10.96.0.0" "172.18.0.2" "172.18.0.2" "172.18.0.2" "172.18.0.22" "172.18.0.2" "172.18.0.2"] on key: "/registry/pods/kube-system/kube-apiserver-kind-control-plane"
IPv4 addresses found ["172.18.0.2" "127.0.0.1" "127.0.0.1" "127.0.0.1" "172.18.0.22" "172.18.0.2" "172.18.0.2"] on key: "/registry/pods/kube-system/kube-scheduler-kind-control-plane"
IPv4 addresses found ["0.0.0.0" "10.244.0.0"] on key: "/registry/configmaps/kube-system/kube-proxy"
IPv4 addresses found ["10.96.0.10" "10.96.0.10"] on key: "/registry/services/specs/kube-system/kube-dns"
IPv4 addresses found ["172.18.0.2"] on key: "/registry/endpointslices/default/kubernetes"
IPv4 addresses found ["10.96.0.10" "127.0.0.1"] on key: "/registry/configmaps/kube-system/kubelet-config-1.22"
IPv4 addresses found ["172.18.0.2"] on key: "/registry/services/endpoints/default/kubernetes"
IPv4 addresses found ["127.0.0.1" "10.244.0.0" "10.96.0.0"] on key: "/registry/configmaps/kube-system/kubeadm-config"
IPv4 addresses found ["10.96.0.1" "10.96.0.1"] on key: "/registry/services/specs/default/kubernetes"
...
```
