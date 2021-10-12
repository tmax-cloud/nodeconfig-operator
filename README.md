# NodeConfig-Operator

[![Go Report Card](https://goreportcard.com/badge/github.com/tmax-cloud/cicd-operator)](https://goreportcard.com/report/github.com/tmax-cloud/nodeconfig-operator)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/tmax-cloud/nodeconfig-operator)
![GitHub](https://img.shields.io/github/license/tmax-cloud/nodeconfig-operator)

![NodeConfig Operator Architecture](docs/figures/NCO_architecture.png)

The NodeConfig Operator implements a Kubernetes API for creating linux host config. It creates cloud-init data and also creates a BareMetalHost if it doesn't exist. 

## Guide
- [Installation Guide](./docs/installation.md)

## Documents
- [NodeConfig API Documentation](docs/api.md)
- [NodeConfig Operator Workflow](docs/nodeconfig-workflow.md)
- [BMO API Documentation](https://github.com/metal3-io/baremetal-operator/blob/capm3-v0.3.2/docs/api.md)
- [BMO Configuration](https://github.com/metal3-io/baremetal-operator/blob/capm3-v0.3.2/docs/configuration.md)
