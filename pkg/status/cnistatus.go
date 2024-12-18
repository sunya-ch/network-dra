package status

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cnitypes "github.com/containernetworking/cni/pkg/types"
	cni100 "github.com/containernetworking/cni/pkg/types/100"
	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type CNIStatusHandler struct {
	ClientSet clientset.Interface
}

type CNIResultList struct {
	Results []interface{} `json:"results"`
}

func (cnish *CNIStatusHandler) UpdateStatus(ctx context.Context, claim *resourcev1beta1.ResourceClaim, result cnitypes.Result) error {
	var resultList CNIResultList
	var networkData *resourcev1beta1.NetworkDeviceData
	if len(claim.Status.Devices) > 0 {
		var prevList CNIResultList
		if err := json.Unmarshal(claim.Status.Devices[0].Data.Raw, &prevList); err == nil {
			resultList.Results = append(prevList.Results, result)
		} else {
			klog.Infof("failed to unmarshal previous list: %v", err)
		}
		networkData = claim.Status.Devices[0].NetworkData
	}
	if len(resultList.Results) == 0 {
		resultList = CNIResultList{
			Results: []interface{}{result},
		}
	}
	resultBytes, err := json.Marshal(resultList)
	if err != nil {
		return fmt.Errorf("cni.handleClaim: failed to json.Marshal result (%v): %v", result, err)
	}
	cniResult, err := cni100.NewResultFromResult(result)
	if err != nil {
		return fmt.Errorf("cni.handleClaim: failed to NewResultFromResult result (%v): %v", result, err)
	}
	newNetworkData := cniResultToNetworkData(cniResult)
	if networkData != nil {
		networkData.IPs = append(networkData.IPs, newNetworkData.IPs...)
		if !strings.Contains(networkData.HardwareAddress, newNetworkData.HardwareAddress) {
			networkData.HardwareAddress = networkData.HardwareAddress + "," + newNetworkData.HardwareAddress
		}
		if !strings.Contains(networkData.InterfaceName, newNetworkData.InterfaceName) {
			networkData.InterfaceName = networkData.InterfaceName + "," + newNetworkData.InterfaceName
		}
	} else {
		networkData = newNetworkData
	}
	if len(claim.Status.Devices) == 0 {
		claim.Status.Devices = append(claim.Status.Devices, resourcev1beta1.AllocatedDeviceStatus{
			Driver: claim.Status.Allocation.Devices.Results[0].Driver,
			Pool:   claim.Status.Allocation.Devices.Results[0].Pool,
			Device: claim.Status.Allocation.Devices.Results[0].Device,
			Data: runtime.RawExtension{
				Raw: resultBytes,
			},
			NetworkData: networkData,
		})
	} else {
		claim.Status.Devices[0].Data = runtime.RawExtension{
			Raw: resultBytes,
		}
		// NetworkData has updated by pointer.
	}

	_, err = cnish.ClientSet.ResourceV1beta1().ResourceClaims(claim.GetNamespace()).UpdateStatus(ctx, claim, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("cni.handleClaim: failed to update resource claim status (%v): %v", result, err)
	}
	return nil
}

func cniResultToNetworkData(cniResult *cni100.Result) *resourcev1beta1.NetworkDeviceData {
	networkData := resourcev1beta1.NetworkDeviceData{}

	for _, ip := range cniResult.IPs {
		networkData.IPs = append(networkData.IPs, ip.Address.String())
	}

	for _, ifs := range cniResult.Interfaces {
		// Only pod interfaces can have sandbox information
		if ifs.Sandbox != "" {
			networkData.InterfaceName = ifs.Name
			networkData.HardwareAddress = ifs.Mac
		}
	}

	return &networkData
}
