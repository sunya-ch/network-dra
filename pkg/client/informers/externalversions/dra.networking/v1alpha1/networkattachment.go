/*
Copyright (c) 2024

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

package v1alpha1

import (
	"context"
	time "time"

	dranetworkingv1alpha1 "github.com/LionelJouin/network-dra/api/dra.networking/v1alpha1"
	versioned "github.com/LionelJouin/network-dra/pkg/client/clientset/versioned"
	internalinterfaces "github.com/LionelJouin/network-dra/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/LionelJouin/network-dra/pkg/client/listers/dra.networking/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// NetworkAttachmentInformer provides access to a shared informer and lister for
// NetworkAttachments.
type NetworkAttachmentInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.NetworkAttachmentLister
}

type networkAttachmentInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewNetworkAttachmentInformer constructs a new informer for NetworkAttachment type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewNetworkAttachmentInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredNetworkAttachmentInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredNetworkAttachmentInformer constructs a new informer for NetworkAttachment type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredNetworkAttachmentInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.DraV1alpha1().NetworkAttachments(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.DraV1alpha1().NetworkAttachments(namespace).Watch(context.TODO(), options)
			},
		},
		&dranetworkingv1alpha1.NetworkAttachment{},
		resyncPeriod,
		indexers,
	)
}

func (f *networkAttachmentInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNetworkAttachmentInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *networkAttachmentInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&dranetworkingv1alpha1.NetworkAttachment{}, f.defaultInformer)
}

func (f *networkAttachmentInformer) Lister() v1alpha1.NetworkAttachmentLister {
	return v1alpha1.NewNetworkAttachmentLister(f.Informer().GetIndexer())
}
