/*
Copyright The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	context "context"
	time "time"

	apiscertmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	versioned "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	internalinterfaces "github.com/cert-manager/cert-manager/pkg/client/informers/externalversions/internalinterfaces"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/client/listers/certmanager/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// IssuerInformer provides access to a shared informer and lister for
// Issuers.
type IssuerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() certmanagerv1.IssuerLister
}

type issuerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewIssuerInformer constructs a new informer for Issuer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewIssuerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredIssuerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredIssuerInformer constructs a new informer for Issuer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredIssuerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CertmanagerV1().Issuers(namespace).List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CertmanagerV1().Issuers(namespace).Watch(context.Background(), options)
			},
			ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CertmanagerV1().Issuers(namespace).List(ctx, options)
			},
			WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CertmanagerV1().Issuers(namespace).Watch(ctx, options)
			},
		},
		&apiscertmanagerv1.Issuer{},
		resyncPeriod,
		indexers,
	)
}

func (f *issuerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredIssuerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *issuerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apiscertmanagerv1.Issuer{}, f.defaultInformer)
}

func (f *issuerInformer) Lister() certmanagerv1.IssuerLister {
	return certmanagerv1.NewIssuerLister(f.Informer().GetIndexer())
}
