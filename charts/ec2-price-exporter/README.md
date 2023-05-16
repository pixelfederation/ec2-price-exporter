# ec2-price-exporter

[ec2-price-exporter](https://github.com/pixelfederation/ec2-price-exporter) is a price exporter for spot and on-demand AWS instances.


## Prerequisites

-	Kubernetes 1.21 or later

## Installing the Chart

The chart can be installed as follows:

```console
$ helm repo add ec2-price-exporter https://pixelfederation.github.io/ec2-price-exporter
$ helm --namespace=ec2-price-exporter install ec2-price-exporter pixelfederation/ec2-price-exporter
```

To uninstall/delete the `ec2-price-exporter` deployment:

```console
$ helm uninstall ec2-price-exporter
```
The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

See `values.yaml` for configuration notes. Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

See [README.md](https://github.com/pixelfederation/ec2-price-exporter/) and configmap for posible configuration options.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install ec2-price-exporter pixelfederation/ec2-price-exporter -f values.yaml
```