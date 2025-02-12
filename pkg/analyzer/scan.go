package analyzer

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

//ScanK8sDeployObject :
func ScanK8sDeployObject(kind string, objDataBuf []byte) (common.Resource, error) {
	var podSpecv1 v1.PodTemplateSpec
	var resourceCtx common.Resource
	// var replicaCount int32
	switch kind {
	case "Pod":
		zap.S().Info("evaluating pod")
		// obj := parser.ParsePod(bytes.NewReader(dataBuf))
		// podSpecv1 = obj.Spec
	case "ReplicaSet":
		obj := ParseReplicaSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Labels = obj.GetLabels()
		resourceCtx.Resource.ServiceAccountName = obj.Spec.Template.Spec.ServiceAccountName
		resourceCtx.Resource.Kind = kind
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		// resourceCtx.Namespace = obj.GetNamespace()
		podSpecv1 = obj.Spec.Template
		// resourceCtx.Resource.ReplicaCount = int(*obj.Spec.Replicas)
	case "ReplicationController":
		obj := ParseReplicationController(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.ServiceAccountName = obj.Spec.Template.Spec.ServiceAccountName
		// for k, v := range obj.Spec.Selector.MatchLabels {
		// 	resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		// }
		podSpecv1 = *obj.Spec.Template
		// resourceCtx.Resource.ReplicaCount = int(*obj.Spec.Replicas)
	case "Deployment":
		obj := ParseDeployment(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.ServiceAccountName = obj.Spec.Template.Spec.ServiceAccountName
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		podSpecv1 = obj.Spec.Template
		// resourceCtx.Resource.ReplicaCount = int(*obj.Spec.Replicas)
	case "DaemonSet":
		obj := ParseDaemonSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.ServiceAccountName = obj.Spec.Template.Spec.ServiceAccountName
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		podSpecv1 = obj.Spec.Template
	case "StatefulSet":
		obj := ParseStatefulSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.ServiceAccountName = obj.Spec.Template.Spec.ServiceAccountName
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		podSpecv1 = obj.Spec.Template
	case "Job":
		obj := ParseJob(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.ServiceAccountName = obj.Spec.Template.Spec.ServiceAccountName
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		podSpecv1 = obj.Spec.Template
	default:
		return resourceCtx, fmt.Errorf("unsupported object type: `%s`", kind)
	}

	parseDeployResource(podSpecv1, &resourceCtx)
	return resourceCtx, nil
}

func ScanK8sConfigmapObject(kind string, objDataBuf []byte) (common.CfgMap, error) {
	obj := ParseConfigMap(bytes.NewReader(objDataBuf))

	fullName := obj.ObjectMeta.Namespace + "/" + obj.ObjectMeta.Name
	data := map[string]string{}
	for k, v := range obj.Data {
		isPotentialAddress := identifyAddressValue(v)
		if isPotentialAddress {
			data[k] = v
		}
	}
	return common.CfgMap{FullName: fullName, Data: data}, nil
}

//ScanK8sServiceObject :
func ScanK8sServiceObject(kind string, objDataBuf []byte) (common.Service, error) {
	var svcSpecv1 v1.ServiceSpec
	var serviceCtx common.Service
	switch kind {
	case "Service":
		svcObj := ParseService(bytes.NewReader(objDataBuf))
		serviceCtx.Resource.Name = svcObj.GetName()
		serviceCtx.Resource.Namespace = svcObj.Namespace
		serviceCtx.Resource.Kind = kind
		serviceCtx.Resource.Type = string(svcObj.Spec.Type)
		for k, v := range svcObj.Spec.Selector {
			serviceCtx.Resource.Selectors = append(serviceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		// serviceCtx.Resource.Selectors = svcObj.GetLabels()
		svcSpecv1 = svcObj.Spec
	default:
		return serviceCtx, fmt.Errorf("unsupported object type: `%s`", kind)
	}
	parseServiceResource(svcSpecv1, &serviceCtx)

	return serviceCtx, nil
}

func parseDeployResource(podSpec v1.PodTemplateSpec, resourceCtx *common.Resource) error {
	for _, container := range podSpec.Spec.Containers {
		resourceCtx.Resource.Image.ID = container.Image
		for _, p := range container.Ports {
			n := common.NetworkAttr{}
			n.ContainerPort = int(p.ContainerPort)
			n.HostPort = int(p.HostPort)
			n.Protocol = string(p.Protocol)
			resourceCtx.Resource.Network = append(resourceCtx.Resource.Network, n)
		}
		for _, e := range container.Env {
			if e.Value != "" {
				isPotentialAddress := identifyAddressValue(e.Value)
				if isPotentialAddress {
					resourceCtx.Resource.Envs = append(resourceCtx.Resource.Envs, e.Value)
				}
			} else if e.ValueFrom != nil && e.ValueFrom.ConfigMapKeyRef != nil {
				keyRef := e.ValueFrom.ConfigMapKeyRef
				if keyRef.Name != "" && keyRef.Key != "" { // just store ref for now - check later if it's a network address
					resourceCtx.Resource.ConfigMapKeyRefs = append(resourceCtx.Resource.ConfigMapKeyRefs, common.CfgMapKeyRef{Name: keyRef.Name, Key: keyRef.Key})
				}
			}
		}
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil { // just store ref for now - check later if the config map values contain a network address
				resourceCtx.Resource.ConfigMapRefs = append(resourceCtx.Resource.ConfigMapRefs, envFrom.ConfigMapRef.Name)
			}
		}
	}
	return nil
}

//identifyAddressValue checks if a given string is a potential network address
func identifyAddressValue(value string) bool {
	_, err := url.Parse(value)
	if err != nil {
		return false
	}
	_, err = strconv.Atoi(value)
	return err != nil // we do not accept integers as network addresses
}

func parseServiceResource(svcSpec v1.ServiceSpec, serviceCtx *common.Service) error {
	for _, p := range svcSpec.Ports {
		n := common.SvcNetworkAttr{}
		n.Port = int(p.Port)
		n.TargetPort = p.TargetPort
		n.Protocol = string(p.Protocol)
		serviceCtx.Resource.Network = append(serviceCtx.Resource.Network, n)
	}
	return nil
}
