# Papertrail-go
Papertrail-go is a Go (golang) library built using Go modules for shipping logs to [Papertrail](https://www.papertrail.com/) using syslog over udp or tcp or tls.
> __Warning:__ This is a Go library project. It will **NOT** produce a consumable executable.

## Papertrail
In order to ship logs to Papertrail, you will need a Papertrail account. If you don't have one already, you can sign up for one [here](https://www.papertrail.com/). After you are logged in, you will need to create a `Log Destination` from under the `Settings` menu. When a log destination is created, you will be given a host:port combo.


For any help with Papertrail, please check out their help page [here](https://help.papertrailapp.com/).

## Usage
To get the package:
```
go get github.com/solarwinds/papertrail-go
```
For a detailed usage example, please check out:
https://github.com/solarwinds/cabbage/blob/master/logshipper/papertrail.go

For development, you should be able to clone this repository to any convenient location on your machine.

To run all the static checks:
```
make lint
```

To run tests:
```
make tests
```

# Questions/Comments?
Please [open an issue](https://github.com/solarwinds/papertrail-go/issues/new), we'd love to hear from you. As a SolarWinds Project, it is supported in a best-effort fashion.