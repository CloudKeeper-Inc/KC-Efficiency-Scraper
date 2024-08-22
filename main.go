package main

import (
	"kubecost-efficiency-fetcher/cluster"
	"kubecost-efficiency-fetcher/controller"
	"kubecost-efficiency-fetcher/controllerKind"
	"kubecost-efficiency-fetcher/deployment"
	"kubecost-efficiency-fetcher/configs"
	"kubecost-efficiency-fetcher/namespace"
	"kubecost-efficiency-fetcher/node"
	"kubecost-efficiency-fetcher/pod"
	// "kubecost-efficiency-fetcher/s3Upload"
	"kubecost-efficiency-fetcher/service"
	"sync"
)



func main() {

	wg := &sync.WaitGroup{}
	wg.Add(6)
	
	go cluster.FetchAndWriteClusterData(configs.KubecostEndpoint,configs.ClusterName,configs.Window,configs.BucketName, configs.BucketRegion, wg)
	
	go node.FetchAndWriteNodeData(configs.KubecostEndpoint,configs.ClusterName,configs.Window,configs.BucketName, configs.BucketRegion, wg)

	go pod.FetchAndWritePodData(configs.KubecostEndpoint,configs.ClusterName,configs.Window,configs.BucketName, configs.BucketRegion, wg)

	go namespace.FetchAndWriteNamespaceData(configs.KubecostEndpoint,configs.ClusterName,configs.Window,configs.BucketName, wg)
	
	go service.FetchAndWriteServiceData(configs.KubecostEndpoint,configs.ClusterName,configs.Window,configs.BucketName, configs.BucketRegion, wg)

	go deployment.FetchAndWriteDeploymentData(configs.KubecostEndpoint,configs.ClusterName,configs.Window,configs.BucketName, configs.BucketRegion, wg)

	go controller.FetchAndWriteControllerData(configs.KubecostEndpoint,configs.ClusterName,configs.Window,configs.BucketName, configs.BucketRegion, wg)

	go controllerKind.FetchAndWriteControllerKindData(configs.KubecostEndpoint,configs.ClusterName,configs.Window,configs.BucketName, configs.BucketRegion, wg)

	wg.Wait()

}
