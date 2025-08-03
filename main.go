package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"

	server "github.com/k-ogger/kogger-service/kogger"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	cronjobEnv := os.Getenv("CRONJOB")

	if cronjobEnv == "true" {
		fmt.Println("Running in cronjob mode")
		// runCronJob()
		return
	} else {
		kubeconfig, err := rest.InClusterConfig()
		if err != nil {
			fmt.Errorf("Failed to retrieve in-cluster Kubernetes config: %s", err)
			return
		}

		clientset, err := kubernetes.NewForConfig(kubeconfig)
		if err != nil {
			fmt.Errorf("Failed to initialize Kubernetes client: %s", err)
			return
		}

		server.Clientset = clientset

		fmt.Println("Running in server mode")
		server.Run()
	}
}

// func runCronJob() {
// 	fmt.Println("Starting cronjob execution")
// 	ctx, _ := createContextFromHeader(&http.Request{})
// 	var ok bool
// 	if _, ok = os.LookupEnv("KOGGER_HOST"); !ok {
// 		fmt.Println("ERROR: KOGGER_HOST environment variable is not set")
// 		return
// 	} else {
// 		server.KoggerHost = os.Getenv("KOGGER_HOST")
// 		fmt.Printf("KOGGER_HOST: %s\n", server.KoggerHost)
// 	}
// 	if _, ok = os.LookupEnv("KOGGER_PORT"); !ok {
// 		fmt.Println("ERROR: KOGGER_PORT environment variable is not set")
// 		return
// 	} else {
// 		server.KoggerPort = os.Getenv("KOGGER_PORT")
// 		fmt.Printf("KOGGER_PORT: %s\n", server.KoggerPort)
// 	}

// 	fmt.Printf("Attempting to connect to %v:%v\n", server.KoggerHost, server.KoggerPort)
// 	conn, err := grpc.Dial(fmt.Sprintf("%v:%v", server.KoggerHost, server.KoggerPort), grpc.WithInsecure(), grpc.WithBlock())
// 	if err != nil {
// 		fmt.Printf("ERROR: failed to establish gRPC connection: %v\n", err)
// 		return
// 	}
// 	defer conn.Close()
// 	fmt.Println("gRPC connection established successfully")
// 	client := koggerservicerpc.NewKoggerServiceClient(conn)

// 	pods, err := client.GetResources(ctx, &koggerservicerpc.ResourcesRequest{
// 		Namespace:    "default",
// 		ResourceType: koggerservicerpc.ResourceType_RESOURCE_TYPE_POD,
// 	})
// 	if err != nil {
// 		fmt.Printf("ERROR: failed to get pods: %v\n", err)
// 		return
// 	}
// 	if len(pods.GetResources()) == 0 {
// 		fmt.Println("No pods found")
// 		return
// 	}
// 	for _, pod := range pods.GetResources() {
// 		fmt.Println("########### Processing pod ###########")
// 		fmt.Printf("Pod Name: %s, Namespace: %s\n", pod.Name, pod.Namespace)
// 		fmt.Printf("Pod Status: %s\n", pod.Status)

// 		logs, err := client.GetLogs(ctx, &koggerservicerpc.LogsRequest{
// 			Pod:       pod.Name,
// 			Namespace: pod.Namespace,
// 		})
// 		if err != nil {
// 			fmt.Printf("ERROR: failed to get logs for pod %s in namespace %s: %v\n", pod.Name, pod.Namespace, err)
// 			continue
// 		}
// 		if len(logs.GetEntries()) == 0 {
// 			fmt.Printf("No logs found for pod %s in namespace %s\n", pod.Name, pod.Namespace)
// 			continue
// 		}
// 		for _, log := range logs.GetEntries() {
// 			fmt.Printf("Log Container: %s, Timestamp: %s, Message: %s\n", log.GetContainer(), log.GetTimestamp(), log.GetMessage())
// 		}
// 	}

// 	fmt.Println("Cron job completed successfully")
// }

var (
	grpcTokenAlphabet = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ23456789")
)

func generateGrpcToken() string {
	tk := make([]byte, 16)
	for i := range tk {
		tk[i] = grpcTokenAlphabet[rand.Intn(len(grpcTokenAlphabet))]
	}
	return string(tk)
}

func createContextFromHeader(r *http.Request) (context.Context, string) {
	grpcToken := generateGrpcToken()

	ctx := metadata.AppendToOutgoingContext(r.Context(), "grpc-token", grpcToken)

	return ctx, grpcToken
}
