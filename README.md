# Cabbage

Cabbage is the easiest way to get logs out of your k8s cluster and into any syslog-compatible sink. Because it doesn't require DaemonSets, sidecars, fluentd or persistent claims, it's one of the only solutions for logging in nodeless clusters, such as EKS on Fargate. But it's also perfect for smaller, local dev clusters to setup logging withing seconds.

## Usage

> :warn: You need a pull secret for quay.io/solarwinds in your cluster! If you don't have access to quay, build the image yourself and push it to your registry.

You need to deploy one cabbage per namespace. There are only a few options you need to set before applying the deployments. The default kustomization and patch in the repo root provide an example config for one of our clusters. Please change those values as follows:

1. In `kustomization.yaml` set the target namespace
2. In `logging-config-patch.yaml` follow the comments to setup the connection to the syslog sink (Papertrail in this example) and set a system tag for the syslog messages.

That's it. Preview with `kubectl apply -k . --dry-run -o yaml` and remove `--dry-run` to apply.

## How it works

Cabbage deploys `kail` in an alpine container, using it to query the k8s API for pods (and keeping the pod list in sync) and their logs. Kail is a command line k8s logging client that lives at the opposite end of the specificity spectrum from `kubectl logs ...`. You can run kail yourself by cloning this repo and running `./src/kail` on linux. This will give you all logs from all pods in all namespaces. To run kail on other OS and to learn more about filters, read the [kail usage guide](https://github.com/boz/kail/tree/eb6734178238dc794641e82779855fabc2071e23#usage).

## Security

The default deployment for cabbage will grant the default service account for the given namespace access to the k8s APIs `pods` and `pods/logs`. Technically, other pods will be able to use these capabilities. To further lock this down, edit (or kustomize) the ClusterRoleBinding to bind to a different SA. You will have setup that SA and make sure the cabbage deployment can use it.
