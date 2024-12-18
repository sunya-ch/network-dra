package nri

import (
	"context"
	"fmt"

	"github.com/containerd/nri/pkg/api"
	"github.com/containerd/nri/pkg/stub"
	cniv1 "github.com/kubernetes-sigs/multi-network/pkg/cni/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type Plugin struct {
	Stub      stub.Stub
	ClientSet clientset.Interface
	CNI       *cniv1.CNI
}

func (p *Plugin) RunPodSandbox(ctx context.Context, pod *api.PodSandbox) error {
	klog.FromContext(ctx).Info("RunPodSandbox", "pod.Name", pod.Name)

	podNetworkNamespace := getNetworkNamespace(pod)
	if podNetworkNamespace == "" {
		return fmt.Errorf("error getting network namespace for pod '%s' in namespace '%s'", pod.Name, pod.Namespace)
	}

	// add existing claims
	podObj, err := p.ClientSet.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err == nil {
		claims := podObj.Spec.ResourceClaims
		for _, claim := range claims {
			if claim.ResourceClaimName != nil {
				if claimObj, err := p.ClientSet.ResourceV1beta1().ResourceClaims(pod.Namespace).Get(ctx, *claim.ResourceClaimName, metav1.GetOptions{}); err == nil {
					if added := p.CNI.AddNewPodResource(types.UID(pod.Uid), claimObj); added {
						klog.FromContext(ctx).Info("RunPodSandbox add existing claim", "pod.Name", pod.Name, "claim.Name", *claim.ResourceClaimName)
					}
				}
			}
		}
	}

	err = p.CNI.AttachNetworks(ctx, pod.Id, pod.Uid, pod.Name, pod.Namespace, podNetworkNamespace)
	if err != nil {
		return fmt.Errorf("error CNI.AttachNetworks for pod '%s' (uid: %s) in namespace '%s': %v", pod.Name, pod.Uid, pod.Namespace, err)
	}

	return nil
}

func getNetworkNamespace(pod *api.PodSandbox) string {
	for _, namespace := range pod.Linux.GetNamespaces() {
		if namespace.Type == "network" {
			return namespace.Path
		}
	}

	return ""
}
