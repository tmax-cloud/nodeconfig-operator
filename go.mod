module github.com/tmax-cloud/nodeconfig-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/metal3-io/baremetal-operator/apis v0.0.0-20210416073321-c927d1d8da76
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/pkg/errors v0.9.1
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac // indirect
	golang.org/x/tools v0.1.7 // indirect
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.2
	k8s.io/cluster-bootstrap v0.21.3
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/cluster-api v0.4.0
	sigs.k8s.io/controller-runtime v0.9.1
)
