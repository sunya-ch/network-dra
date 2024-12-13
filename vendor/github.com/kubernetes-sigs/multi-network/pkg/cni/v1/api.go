package v1

import "k8s.io/apimachinery/pkg/runtime"

type Parameters struct {
	Config        runtime.RawExtension `json:"config,omitempty"`
	InterfaceName string               `json:"interface,omitempty"`
}
