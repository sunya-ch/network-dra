package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/containernetworking/cni/libcni"
	cnitypes "github.com/containernetworking/cni/pkg/types"
	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type PodResourceStore interface {
	Get(podUID types.UID) []*resourcev1beta1.ResourceClaim
	Delete(podUID types.UID)
}

type UpdateStatus func(ctx context.Context, claim *resourcev1beta1.ResourceClaim, cniResult cnitypes.Result) error

type CNI struct {
	podResourceStore PodResourceStore
	cniConfig        *libcni.CNIConfig
	driverName       string
	updateStatusFunc UpdateStatus
}

func New(
	driverName string,
	chrootDir string,
	cniPath []string,
	cniCacheDir string,
	updateStatusFunc UpdateStatus,
	podResourceStore PodResourceStore,
) *CNI {
	exec := &chrootExec{
		Stderr:    os.Stderr,
		ChrootDir: chrootDir,
	}

	cni := &CNI{
		podResourceStore: podResourceStore,
		cniConfig:        libcni.NewCNIConfigWithCacheDir(cniPath, cniCacheDir, exec),
		driverName:       driverName,
		updateStatusFunc: updateStatusFunc,
	}

	return cni
}

func (cni *CNI) AttachNetworks(
	ctx context.Context,
	podSandBoxID string,
	podUID string,
	podName string,
	podNamespace string,
	podNetworkNamespace string,
) error {
	claims := cni.podResourceStore.Get(types.UID(podUID))

	klog.Infof("cni.AttachNetworks: attach networks on pod %s (%s)", podName, podUID)

	for _, claim := range claims {
		err := cni.handleClaim(
			ctx,
			podSandBoxID,
			podUID,
			podName,
			podNamespace,
			podNetworkNamespace,
			claim,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cni *CNI) handleClaim(
	ctx context.Context,
	podSandBoxID string,
	podUID string,
	podName string,
	podNamespace string,
	podNetworkNamespace string,
	claim *resourcev1beta1.ResourceClaim,
) error {
	if claim.Status.Allocation == nil ||
		len(claim.Status.Allocation.Devices.Results) != 1 ||
		len(claim.Status.Allocation.Devices.Config) != 1 ||
		claim.Status.Allocation.Devices.Results[0].Driver != cni.driverName ||
		claim.Status.Allocation.Devices.Config[0].Opaque == nil ||
		claim.Status.Allocation.Devices.Config[0].Opaque.Driver != cni.driverName {
		return nil
	}

	klog.Infof("cni.handleClaim: attach network (claim: %s) on pod %s (%s)", claim.Name, podName, podUID)

	cniParameters := &Parameters{}
	err := json.Unmarshal(claim.Status.Allocation.Devices.Config[0].Opaque.Parameters.Raw, cniParameters)
	if err != nil {
		return fmt.Errorf("cni.handleClaim: failed to json.Unmarshal Opaque.Parameters: %v", err)
	}

	result, err := cni.add(
		ctx,
		podSandBoxID,
		podUID,
		podName,
		podNamespace,
		podNetworkNamespace,
		cniParameters,
	)
	if err != nil {
		return err
	}

	if cni.updateStatusFunc != nil {
		err = cni.updateStatusFunc(ctx, claim, result)
		if err != nil {
			return fmt.Errorf("cni.handleClaim: failed to update status (%v): %v", result, err)
		}
	}

	return nil
}

func (cni *CNI) add(
	ctx context.Context,
	podSandBoxID string,
	podUID string,
	podName string,
	podNamespace string,
	podNetworkNamespace string,
	parameters *Parameters,
) (cnitypes.Result, error) {
	rt := &libcni.RuntimeConf{
		ContainerID: podSandBoxID,
		NetNS:       podNetworkNamespace,
		IfName:      parameters.InterfaceName,
		Args: [][2]string{
			{"IgnoreUnknown", "true"},
			{"K8S_POD_NAMESPACE", podNamespace},
			{"K8S_POD_NAME", podName},
			{"K8S_POD_INFRA_CONTAINER_ID", podSandBoxID},
			{"K8S_POD_UID", podUID},
		},
	}

	confList, err := libcni.ConfListFromBytes(parameters.Config.Raw)
	if err != nil {
		return nil, fmt.Errorf("cni.add: failed to ConfListFromBytes: %v", err)
	}

	result, err := cni.cniConfig.AddNetworkList(ctx, confList, rt)
	if err != nil {
		return nil, fmt.Errorf("cni.add: failed to AddNetwork: %v", err)
	}

	return result, nil
}

func (cni *CNI) DetachNetworks(
	ctx context.Context,
	podSandBoxID string,
	podUID string,
	podName string,
	podNamespace string,
	podNetworkNamespace string,
) error {
	// TODO
	return nil
}
