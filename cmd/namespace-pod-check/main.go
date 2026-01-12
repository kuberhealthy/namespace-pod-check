package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kuberhealthy/kuberhealthy/v3/pkg/checkclient"
	nodecheck "github.com/kuberhealthy/kuberhealthy/v3/pkg/nodecheck"
	log "github.com/sirupsen/logrus"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// podName is the test pod name created in each namespace.
	podName = "kuberhealthy-namespace-checker-pod"
)

// main loads configuration and validates pod creation across namespaces.
func main() {
	// Enable nodecheck debug output for parity with v2 behavior.
	nodecheck.EnableDebugOutput()

	// Create a timeout context for readiness checks.
	checkTimeLimit := time.Minute * 1
	ctx, _ := context.WithTimeout(context.Background(), checkTimeLimit)

	// Create a Kubernetes client.
	kubernetesClient, err := createKubeClient()
	if err != nil {
		reportFailureAndExit(fmt.Errorf("error creating kube client: %w", err))
		return
	}

	// Wait for Kuberhealthy to be reachable before running the check.
	err = nodecheck.WaitForKuberhealthy(ctx)
	if err != nil {
		log.Errorln("Error waiting for kuberhealthy endpoint to be contactable by checker pod with error:", err.Error())
	}

	// List namespaces.
	namespaces, err := listNamespaces(ctx, kubernetesClient)
	if err != nil {
		reportFailureAndExit(fmt.Errorf("failed to list namespaces: %w", err))
		return
	}
	log.Infoln("Found", len(namespaces.Items), "namespaces")

	// Track pod results across namespaces.
	successfulPods := 0
	failedPods := 0

	// Create and delete a test pod in each namespace.
	for _, namespace := range namespaces.Items {
		log.Infoln("DEPLOYING POD IN NAMESPACE", namespace.Name)

		err = deployPod(ctx, namespace.Name, podName, kubernetesClient)
		if err != nil {
			log.Error(err)
			failedPods++
			continue
		}

		err = deletePod(ctx, namespace.Name, podName, kubernetesClient)
		if err != nil {
			log.Error(err)
			failedPods++
			continue
		}

		successfulPods++
	}

	// Report a failure when any namespace fails.
	if failedPods != 0 {
		reportErr := fmt.Errorf("namespace-pod-check was unable to deploy or delete test pods in %s out of %s namespaces", strconv.Itoa(failedPods), strconv.Itoa(len(namespaces.Items)))
		reportFailureAndExit(reportErr)
		return
	}

	// Report success when all namespaces succeed.
	log.Infoln("namespace-pod-check was able to successfully deploy and delete test pods in", successfulPods, "namespaces")
	err = checkclient.ReportSuccess()
	if err != nil {
		log.Fatalln("error when reporting to kuberhealthy with error:", err)
	}
	log.Infoln("Successfully reported to kuberhealthy.")
}

// listNamespaces returns all namespaces using the provided client.
func listNamespaces(ctx context.Context, client *kubernetes.Clientset) (*core.NamespaceList, error) {
	// List namespaces without filters.
	listOpts := metav1.ListOptions{}
	return client.CoreV1().Namespaces().List(ctx, listOpts)
}

// deployPod creates a test pod in the given namespace.
func deployPod(ctx context.Context, namespace string, name string, client *kubernetes.Clientset) error {
	// Build the test pod specification.
	pod := getPodObject(name, namespace)
	createOptions := metav1.CreateOptions{}

	// Create the pod in the cluster.
	_, err := client.CoreV1().Pods(namespace).Create(ctx, pod, createOptions)
	if err != nil {
		return fmt.Errorf("error deploying pod %s in namespace %s: %w", name, namespace, err)
	}
	log.Infoln("Pod", name, "created successfully in namespace:", namespace)
	return nil
}

// deletePod deletes a test pod in the given namespace.
func deletePod(ctx context.Context, namespace string, name string, client *kubernetes.Clientset) error {
	// Delete the pod from the cluster.
	deleteOptions := metav1.DeleteOptions{}
	err := client.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		return fmt.Errorf("error deleting pod %s in namespace %s: %w", name, namespace, err)
	}
	log.Infoln("Pod", name, "successfully deleted in namespace:", namespace)
	return nil
}

// getPodObject returns the pod specification for the test pod.
func getPodObject(name string, namespace string) *core.Pod {
	// Define a simple busybox pod.
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "demo",
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            "busybox",
					Image:           "busybox",
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"sleep",
						"3600",
					},
				},
			},
		},
	}
}

// reportFailureAndExit reports an error to Kuberhealthy and exits.
func reportFailureAndExit(err error) {
	// Report the failure to Kuberhealthy.
	reportErr := checkclient.ReportFailure([]string{err.Error()})
	if reportErr != nil {
		log.Fatalln("error when reporting to kuberhealthy with error:", reportErr)
	}
	log.Infoln("Successfully reported error to kuberhealthy")
	os.Exit(0)
}
