# rkubelog

rkubelog is the easiest way to get logs out of your k8s cluster and into [Papertrail](https://www.papertrail.com/) and [Loggly](https://www.loggly.com/). Because it doesn't require DaemonSets, sidecars, fluentd or persistent claims, it's one of the only solutions for logging in nodeless clusters, such as EKS on Fargate. But it's also perfect for smaller, local dev clusters to setup logging within seconds.

## Usage

> __Info:__ Make sure to always reference rkubelog versions explicitly in the image. Do not use `latest` tags. The current version is `quay.io/solarwinds/rkubelog:<github_version>`, where `github_version` is the latest revision listed in [Releases](https://github.com/solarwinds/rkubelog/releases), for example `r13`.

By default, rkubelog runs in the `kube-system` namespace and will observe all logs from all pods in all namespaces except from itself or any other service in `kube-system`.

In `logging-config-patch.yaml` follow the comments to setup the connection to the syslog sink (Papertrail in this example) and set a system tag for the syslog messages.

That's it. Preview with `kubectl apply -k . --dry-run -o yaml` and remove `--dry-run` to apply.

## How it works

rkubelog deploys a customized `kail` in an alpine container, using it to query the k8s API for pods (and keeping the pod list in sync) and their logs. Kail is a command line k8s logging client that lives at the opposite end of the specificity spectrum from `kubectl logs ...`. You can run kail yourself by cloning this repo and running `go run main.go`. This will give you all logs from all pods in all namespaces. 
To learn more about filters, read the [kail usage guide](https://github.com/boz/kail/tree/eb6734178238dc794641e82779855fabc2071e23#usage).

### Papertrail
In order to ship logs to Papertrail, you will need a Papertrail account. If you don't have one already, you can sign up for one [here](https://www.papertrail.com/). After you are logged in, you will need to create a `Log Destination` from under the `Settings` menu. When a log destination is created, you will be given a host:port combo.

**Environment variables**
- `PAPERTRAIL_PROTOCOL` - Acceptable values are udp, tcp, tls. This also depends on the choices that are selected under the `Destination Settings`; by default, a new destination accepts TLS and UDP connections.
- `PAPERTRAIL_HOST` - Log destination host
- `PAPERTRAIL_PORT` - Log destination port

You can update the `logging-secrets.yaml` and `logging-config-patch.yaml` files accordingly with these values, remove the unused ones and use `kubectl apply...` as described above.

For any help with Papertrail, please check out their help page [here](https://help.papertrailapp.com/).

Apart from the environment variables, we have also added some extra flags to help configure the Papertrail log shipper.
`--pt-db-location` - Location for temporary db used by Papertrail shipper. Default value is "./db".
`--pt-data-retention` - Retention period for local Papertrail log data. Default value is "4h".
`--pt-worker-count` - Papertrail log shipper worker count per CPU. Default value is 10. If there are 4 CPUs, the total will be 40.
`--pt-max-disk-usage` - Papertrail log shipper max disk usage in percent. Default value is 50.

If you want to override the defaults for any of the above flags, they will have to be passed in as arguments to main process.

### Loggly
In order to ship logs to Loggly, you will need a Loggly account. If you don't have one already you can sign up for one [here](https://www.loggly.com/). After you are logged in, you will need to create a `Customer Token` from under the `Source Setup` menu item.

**Environment variable**
- `LOGGLY_TOKEN` - customer token from Loggly (__not__ API token)

For any help with Loggly, please checkout their help page [here](https://www.loggly.com/docs-index/).

You can update the `logging-secrets.yaml` and `logging-config-patch.yaml` files accordingly with these values, remove the unused ones and use `kubectl apply...` as described above.

## Development

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

# Feedback

Please [open an issue](https://github.com/solarwinds/rkubelog/issues/new), we'd love to hear from you. As a SolarWinds Project, it is supported in a best-effort fashion.

# Security

If you have identified a security vulnerability, please send an email to infosec@solarwinds.com (monitored 24/7). Please do not open a public issue.
