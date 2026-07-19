/*
 * Copyright 2024 the KubeSphere Authors.
 * Please refer to the LICENSE file in the root directory of the project.
 * https://github.com/kubesphere/kubesphere/blob/master/LICENSE
 */

package ingress

import (
	"context"
	"strconv"
	"strings"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/apiserver/query"
	"kubesphere.io/kubesphere/pkg/models/resources/v1alpha3"
)

const (
	// fieldPathType allows filtering ingresses by path type.
	// Usage: ?fieldSelector=pathType=Prefix
	fieldPathType = "pathType"
)

// SupportedPathTypes is the canonical list of Kubernetes Ingress path types
// that KubeSphere exposes.  All three values are accepted by the Kubernetes API;
// the UI should present all of them so users can choose appropriately.
var SupportedPathTypes = []v1.PathType{
	v1.PathTypeExact,
	v1.PathTypePrefix,
	v1.PathTypeImplementationSpecific,
}

// IngressPath is a flattened representation of a single HTTP path rule inside
// an Ingress object, including the host it belongs to, the path, the path type,
// and the backend service details.  Returning this view makes it easy for the
// UI to display and edit individual routing entries.
type IngressPath struct {
	// Host is the Ingress rule host (empty string means "any host").
	Host string `json:"host"`
	// Path is the HTTP URI pattern.
	Path string `json:"path"`
	// PathType is one of Exact, Prefix, or ImplementationSpecific.
	PathType v1.PathType `json:"pathType"`
	// ServiceName is the backend Service name.
	ServiceName string `json:"serviceName"`
	// ServicePort is the backend Service port number or name.
	ServicePort string `json:"servicePort"`
}

type ingressGetter struct {
	cache runtimeclient.Reader
}

func New(cache runtimeclient.Reader) v1alpha3.Interface {
	return &ingressGetter{cache: cache}
}

func (g *ingressGetter) Get(namespace, name string) (runtime.Object, error) {
	ingress := &v1.Ingress{}
	return ingress, g.cache.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, ingress)
}

func (g *ingressGetter) List(namespace string, query *query.Query) (*api.ListResult, error) {
	ingresses := &v1.IngressList{}
	if err := g.cache.List(context.Background(), ingresses, client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: query.Selector()}); err != nil {
		return nil, err
	}
	var result []runtime.Object
	for _, item := range ingresses.Items {
		result = append(result, item.DeepCopy())
	}
	return v1alpha3.DefaultList(result, query, g.compare, g.filter), nil
}

func (g *ingressGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {
	leftIngress, ok := left.(*v1.Ingress)
	if !ok {
		return false
	}

	rightIngress, ok := right.(*v1.Ingress)
	if !ok {
		return false
	}

	switch field {
	case query.FieldUpdateTime:
		fallthrough
	default:
		return v1alpha3.DefaultObjectMetaCompare(leftIngress.ObjectMeta, rightIngress.ObjectMeta, field)
	}
}

func (g *ingressGetter) filter(object runtime.Object, filter query.Filter) bool {
	ingress, ok := object.(*v1.Ingress)
	if !ok {
		return false
	}
	switch filter.Field {
	case fieldPathType:
		return ingressHasPathType(ingress, v1.PathType(filter.Value))
	default:
		return v1alpha3.DefaultObjectMetaFilter(ingress.ObjectMeta, filter)
	}
}

// ingressHasPathType returns true if any HTTP path rule in the ingress uses the
// specified pathType.  The comparison is case-insensitive to be lenient with
// client input.
func ingressHasPathType(ingress *v1.Ingress, want v1.PathType) bool {
	wantLower := strings.ToLower(string(want))
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			if p.PathType == nil {
				continue
			}
			if strings.ToLower(string(*p.PathType)) == wantLower {
				return true
			}
		}
	}
	return false
}

// ListIngressPaths expands all HTTP path rules in an Ingress into a flat slice
// of IngressPath entries.  Each entry carries the host, path, pathType, and
// backend service details, giving callers a straightforward view of every
// routing rule without needing to navigate the nested Ingress spec.
func ListIngressPaths(ingress *v1.Ingress) []IngressPath {
	var result []IngressPath
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			ip := IngressPath{
				Host: rule.Host,
				Path: p.Path,
			}
			if p.PathType != nil {
				ip.PathType = *p.PathType
			} else {
				ip.PathType = v1.PathTypeImplementationSpecific
			}
			if p.Backend.Service != nil {
				ip.ServiceName = p.Backend.Service.Name
				// Prefer named port; fall back to numeric port.
				if p.Backend.Service.Port.Name != "" {
					ip.ServicePort = p.Backend.Service.Port.Name
				} else {
					ip.ServicePort = strconv.FormatInt(int64(p.Backend.Service.Port.Number), 10)
				}
			}
			result = append(result, ip)
		}
	}
	return result
}

