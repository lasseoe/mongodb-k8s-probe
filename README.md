# MongoDB Kubernetes Probe

![build](https://github.com/lasseoe/mongodb-k8s-probe/actions/workflows/ci.yml/badge.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

mongodb-k8s-probe is a super lightweight, fast and simple replacement for mongosh startup, readiness and liveness probes in Bitnami MongoDB Helm charts.

The mongosh application is written in Node.js, and although compiled as a binary it's very slow to start, consumes a lot of resources for each and every readiness and liveness check. I've seen anywhere between x2 and x8 CPU & memory usage on an idle replicaset, and occasional timeouts. To Top it off, mongosh by default also send telemetry data, which can slow it down, in particular if there's no Internet egress.

## Installation

Precompiled binaries are available in the [Releases section](https://github.com/lasseoe/mongodb-k8s-probe/releases)

##### Options

 - `-db <database name>` - name of database to connect to, default is `admin` [optional, string]
 - `-host <hostname/IP>` - name or IP of host to connect to, default is `127.0.0.1` [optional, string]
 - `-tls` - connect using mTLS, default is `false` [optional, bool]
 - `-hello` - readiness & startup probe, default is `false` [required, bool]
 - `-ping` - liveness probe, default is `false` [required, bool]
 - `-version` - display version and build information [optional, bool]

The `-hello` and `-ping` options are mutually exclusive, you can't use both, but you **must** use one or the other.

## Example

```yaml
startupProbe:
  enabled: false

readinessProbe:
  enabled: false

livenessProbe:
  enabled: false

customStartupProbe:
  initialDelaySeconds: 20
  failureThreshold: 30
  periodSeconds: 5
  successThreshold: 1
  timeoutSeconds: 5
  exec:
    command:
      - /custom/mongodb-k8s-probe
      - -hello

customReadinessProbe:
  failureThreshold: 6
  periodSeconds: 10
  successThreshold: 1
  timeoutSeconds: 5
  exec:
    command:
      - /custom/mongodb-k8s-probe
      - -hello

customLivenessProbe:
  failureThreshold: 6
  periodSeconds: 10
  successThreshold: 1
  timeoutSeconds: 10
  exec:
    command:
      - /custom/mongodb-k8s-probe
      - -ping

initContainers:
  - name: download-mongodb-k8s-probe
    image: busybox:1.36
    imagePullPolicy: Always
    command:
      - sh
      - -c
      - |
        #!/usr/bin/env bash -e
        wget -O /custom/mongodb-k8s-probe.tar.gz  \
        "https://github.com/lasseoe/mongodb-k8s-probe/releases/download/vX.X.X/mongodb-k8s-probe_linux_amd64.tar.gz"
        tar -C /custom -xzvf /custom/mongodb-k8s-probe.tar.gz mongodb-k8s-probe
        rm /custom/mongodb-k8s-probe.tar.gz
        chmod +x /custom/mongodb-k8s-probe
    volumeMounts:
      - mountPath: "/custom"
        name: mongodb-probe-volume

extraVolumeMounts:
  - name: mongodb-probe-volume
    mountPath: /custom

extraVolumes:
  - name: mongodb-probe-volume
    emptyDir:
      sizeLimit: 20Mi
```

You may want to add securityContext and resources sections to your production YAML.


## TODO

 - [ ] adopt command-line options from mongosh
 - [ ] command-line options for certificate paths
 - [ ] command-line options for hostname, port etc.
 - [ ] environment variables for all options
 - [ ] replicaset checks instead of, or in addition to, ping() and hello()

## MIT License

Copyright (c) 2024 Lasse Østerild

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

