package statefulset

import (
	"github.com/boz/kcache/client"
	"k8s.io/client-go/kubernetes"
)

const resourceName = "statefulsets"

func NewClient(cs kubernetes.Interface, ns string) client.Client {
	scope := cs.AppsV1()
	return client.ForResource(scope.RESTClient(), resourceName, ns)
}
