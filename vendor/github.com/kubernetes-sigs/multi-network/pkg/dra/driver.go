package dra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/klog/v2"
	drapb "k8s.io/kubelet/pkg/apis/dra/v1alpha4"
)

type PodResourceStore interface {
	Add(podUID types.UID, allocation *resourcev1beta1.ResourceClaim)
}

type Driver struct {
	driverName       string
	kubeClient       kubernetes.Interface
	draPlugin        kubeletplugin.DRAPlugin
	podResourceStore PodResourceStore
}

func Start(
	ctx context.Context,
	driverName string,
	nodeName string,
	kubeClient kubernetes.Interface,
	podResourceStore PodResourceStore,
) (*Driver, error) {
	d := &Driver{
		driverName:       driverName,
		kubeClient:       kubeClient,
		podResourceStore: podResourceStore,
	}

	pluginRegistrationPath := filepath.Join("/var/lib/kubelet/plugins_registry/", fmt.Sprintf("%s.sock", driverName))
	driverPluginPath := filepath.Join("/var/lib/kubelet/plugins/", driverName)

	err := os.MkdirAll(driverPluginPath, 0750)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin path %s: %v", driverPluginPath, err)
	}

	driverPluginSocketPath := filepath.Join(driverPluginPath, "plugin.sock")

	opts := []kubeletplugin.Option{
		kubeletplugin.DriverName(driverName),
		kubeletplugin.NodeName(nodeName),
		kubeletplugin.KubeClient(kubeClient),
		kubeletplugin.RegistrarSocketPath(pluginRegistrationPath),
		kubeletplugin.PluginSocketPath(driverPluginSocketPath),
		kubeletplugin.KubeletPluginSocketPath(driverPluginSocketPath),
	}
	driver, err := kubeletplugin.Start(ctx, []interface{}{d}, opts...)
	if err != nil {
		return nil, fmt.Errorf("start kubelet plugin: %w", err)
	}
	d.draPlugin = driver

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 30*time.Second, true, func(context.Context) (bool, error) {
		status := d.draPlugin.RegistrationStatus()
		if status == nil {
			return false, nil
		}
		return status.PluginRegistered, nil
	})
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Driver) Stop() {
	if d.draPlugin != nil {
		d.draPlugin.Stop()
	}
}

func (d *Driver) NodePrepareResources(ctx context.Context, request *drapb.NodePrepareResourcesRequest) (*drapb.NodePrepareResourcesResponse, error) {
	if request == nil {
		return nil, nil
	}
	resp := &drapb.NodePrepareResourcesResponse{
		Claims: make(map[string]*drapb.NodePrepareResourceResponse),
	}

	for uid, claimReq := range request.GetClaims() {
		klog.Infof("NodePrepareResources: Claim Request (%d) %#v", uid, claimReq)
		devices, err := d.nodePrepareResource(ctx, claimReq)
		if err != nil {
			resp.Claims[claimReq.UID] = &drapb.NodePrepareResourceResponse{
				Error: err.Error(),
			}
		} else {
			resp.Claims[claimReq.UID] = &drapb.NodePrepareResourceResponse{
				Devices: devices,
			}
		}
	}
	return resp, nil

}

func (d *Driver) nodePrepareResource(ctx context.Context, claimReq *drapb.Claim) ([]*drapb.Device, error) {
	// The plugin must retrieve the claim itself to get it in the version that it understands.
	claim, err := d.kubeClient.ResourceV1beta1().ResourceClaims(claimReq.Namespace).Get(ctx, claimReq.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("retrieve claim %s/%s: %w", claimReq.Namespace, claimReq.Name, err)
	}
	if claim.Status.Allocation == nil {
		return nil, fmt.Errorf("claim %s/%s not allocated", claimReq.Namespace, claimReq.Name)
	}
	if claim.UID != types.UID(claim.UID) {
		return nil, fmt.Errorf("claim %s/%s got replaced", claimReq.Namespace, claimReq.Name)
	}

	for _, reserved := range claim.Status.ReservedFor {
		if reserved.Resource != "pods" || reserved.APIGroup != "" {
			klog.Infof("claim reference unsupported for %#v", reserved)
			continue
		}

		klog.Infof("nodePrepareResource: Claim Request (%s) reserved for pod %s (%s)", claimReq.UID, reserved.Name, reserved.UID)
		d.podResourceStore.Add(reserved.UID, claim)
	}

	var devices []*drapb.Device
	for _, result := range claim.Status.Allocation.Devices.Results {
		device := &drapb.Device{
			PoolName:   result.Pool,
			DeviceName: result.Device,
		}
		devices = append(devices, device)
	}

	klog.Infof("nodePrepareResource: Devices for Claim Request (%s) %#v", claimReq.UID, devices)

	return devices, nil
}

func (d *Driver) NodeUnprepareResources(ctx context.Context, request *drapb.NodeUnprepareResourcesRequest) (*drapb.NodeUnprepareResourcesResponse, error) {
	if request == nil {
		return nil, nil
	}
	resp := &drapb.NodeUnprepareResourcesResponse{
		Claims: make(map[string]*drapb.NodeUnprepareResourceResponse),
	}

	for _, claimReq := range request.Claims {
		err := d.nodeUnprepareResource(ctx, claimReq)
		if err != nil {
			klog.Infof("error unpreparing ressources for claim %s/%s : %v", claimReq.Namespace, claimReq.Name, err)
			resp.Claims[claimReq.UID] = &drapb.NodeUnprepareResourceResponse{
				Error: err.Error(),
			}
		} else {
			resp.Claims[claimReq.UID] = &drapb.NodeUnprepareResourceResponse{}
		}
	}

	return resp, nil
}

func (d *Driver) nodeUnprepareResource(_ context.Context, _ *drapb.Claim) error {
	// TODO
	return nil
}
