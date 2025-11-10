package util

import (
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//GetKubeConfig : Function to fetch kubeconfig from the path
func GetKubeConfig(kubeConfigPath string) (*rest.Config, error) {
	if kubeConfigPath == "" {
		// Use in-cluster configuration if kubeConfigPath is not specified
		return rest.InClusterConfig()
	}
	// Use specified kubeConfigPath file if provided
	return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
}

//GetNamespaceFromEnv : Function to get namespace from env variable
func GetNamespaceFromEnv() (string, error) {
	podNamespace, present := os.LookupEnv("POD_NAMESPACE")
	if present {
		return podNamespace, nil
	}
	return "", fmt.Errorf("POD_NAMESPACE variable not set")
}
