# rkubelog

rkubelog is the easiest way to get logs out of your k8s cluster and into [Papertrail](https://www.papertrail.com/) and [Loggly](https://www.loggly.com/). Because it doesn't require DaemonSets, sidecars, fluentd or persistent claims, it's one of the only solutions for logging in nodeless clusters, such as EKS on Fargate. But it's also perfect for smaller, local dev clusters to setup logging within seconds.

## Usage

> __Info:__ Make sure to always reference rkubelog versions explicitly in the image. Do not use `latest` tags. The current version is `ghcr.io/solarwinds/rkubelog:<github_version>`, where `github_version` is the latest revision listed in [Releases](https://github.com/solarwinds/rkubelog/releases), for example `r17`.

By default, rkubelog runs in the `kube-system` namespace and will observe all logs from all pods in all namespaces except from itself or any other service in `kube-system`.

To deploy rkubelog:

- Follow the account setup steps in the _How it Works_ section for the logging service of your choice
- Preview the deployment using `kubectl apply -k . --dry-run -o yaml`
- If all looks good, apply the deployment using `kubectl apply -k .`


If you run into issues, please read the _Troubleshooting_ section at the end of this document.

### Static Tags
We have recently added the ability to set static tags which will get sent with all the logs. 

If you are using several instances of `rKubeLog`, you can use this `TAGS` field to differentiate between the logs from the different environments.

If you are interested in using this capability, you can set the value for `TAGS` in the `logging-config-patch.yaml` file. The value can be any string.


### Using Papertrail

In order to ship logs to Papertrail, you will need a Papertrail account. If you don't have one already, you can sign up for one [here](https://www.papertrail.com/). After you are logged in, you will need to create a `Log Destination` from under the `Settings` menu. When a log destination is created, you will be given a host:port combo.

The Papertrail credentials are automatically pulled from a secret named 'logging-secret'. Before deploying rkubelog, you need to [create a kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) with that name in the `kube-system` namespace with the following fields:

- `PAPERTRAIL_PROTOCOL` - Acceptable values are udp, tcp, tls. This also depends on the choices that are selected under the `Destination Settings`; by default, a new destination accepts TLS and UDP connections.
- `PAPERTRAIL_HOST` - Log destination host
- `PAPERTRAIL_PORT` - Log destination port
- `LOGGLY_TOKEN` set to `` to disable Loggly

For any help with Papertrail, please check out their help page [here](https://documentation.solarwinds.com/en/Success_Center/papertrail/Content/papertrail_Documentation.htm).

### Using Loggly

In order to ship logs to Loggly, you will need a Loggly account. If you don't have one already you can sign up for one [here](https://www.loggly.com/). After you are logged in, you will need to create a `Customer Token` from under the `Source Setup` menu item.

The Loggly credentials are automatically pulled from a secret named 'logging-secret'. Before deploying rkubelog, you need to [create a kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) with that name in the `kube-system` namespace with the following fields:

- `LOGGLY_TOKEN` - customer token from Loggly (__not__ API token)

Also add these default values to disable Papertrail logging (following are just samples, please use your account specific values):

- `PAPERTRAIL_PROTOCOL=`
- `PAPERTRAIL_HOST=`
- `PAPERTRAIL_PORT=`

For any help with Loggly, please checkout their help page [here](https://documentation.solarwinds.com/en/Success_Center/loggly/).


__Please note:__ If you want to send logs to both Loggly and Papertrail, you can configure both Loggly and Papertrail related values above to valid ones. Most will only want to use one or the other.

## Development

> __Info:__ You only need to reference this section if you plan to contribute to the rkubelog development.

You will need Go (1.11+) installed on your machine. Then, clone this repo to a suitable location on your machine and `cd` into it. For quick command access the project includes a Makefile.

To build run:
```
make build
```

To run the code:
```
bin/rkubelog
```

You are free to set the described environment variables or pass run time arguments described above and/or follow [kail usage guide](https://github.com/boz/kail/tree/eb6734178238dc794641e82779855fabc2071e23#usage).

To run all the static checks:
```
make lint
```

To run tests:
```
make tests
```

To create a Docker image:
```
make docker
```

# Troubleshooting

### Logs do not appear in Papertrail/Loggly after deploying rkubelog

If you deploy rkubelog on nodeless clusters, such as EKS on Fargate, you may not see logs flow immediately. Specifically on EKS on Fargate it may take up to 2 minutes for a pod to be fully deployed, as AWS needs to provision Fargate nodes. You can check the progress using:

```bash
kubectl get pods -o wide -n kube-system | grep rkubelog
```

- The "status" should be "Runnig"
- The "node" column should have a proper value (`fargate-ip-192-168-X-X.us-east-2.compute.internal`)
- The "nominated node" column should be empty

If all looks good and you still don't see logs in PT/LG, please open an issue.

### Logs suddenly stopped flowing

Please restart the rkubelog deployment:

```bash
kubectl scale deployment rkubelog --replicas=0 -n kube-system
kubectl scale deployment rkubelog --replicas=1 -n kube-system
```

If the problem persists, please open an issue.

# Feedback

Please [open an issue](https://github.com/solarwinds/rkubelog/issues/new), we'd love to hear from you. As a SolarWinds Project, it is supported in a best-effort fashion.

# Security

If you have identified a security vulnerability, please send an email to infosec@solarwinds.com (monitored 24/7). Please do not open a public issue.
