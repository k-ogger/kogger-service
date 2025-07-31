package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"

	server "github.com/k-ogger/kogger-service/kogger"
	"github.com/k-ogger/kogger-service/koggerservicerpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	cronjobEnv := os.Getenv("CRONJOB")

	if cronjobEnv == "true" {
		fmt.Println("Running in cronjob mode")
		runCronJob()
		return
	} else {
		fmt.Println("Running in server mode")
		server.Run()
	}
}

func runCronJob() {
	fmt.Println("Starting cronjob execution")
	ctx, _ := createContextFromHeader(&http.Request{})
	var ok bool
	if _, ok = os.LookupEnv("KOGGER_HOST"); !ok {
		fmt.Println("ERROR: KOGGER_HOST environment variable is not set")
		return
	} else {
		server.KoggerHost = os.Getenv("KOGGER_HOST")
		fmt.Printf("KOGGER_HOST: %s\n", server.KoggerHost)
	}
	if _, ok = os.LookupEnv("KOGGER_PORT"); !ok {
		fmt.Println("ERROR: KOGGER_PORT environment variable is not set")
		return
	} else {
		server.KoggerPort = os.Getenv("KOGGER_PORT")
		fmt.Printf("KOGGER_PORT: %s\n", server.KoggerPort)
	}

	fmt.Printf("Attempting to connect to %v:%v\n", server.KoggerHost, server.KoggerPort)
	conn, err := grpc.Dial(fmt.Sprintf("%v:%v", server.KoggerHost, server.KoggerPort), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		fmt.Printf("ERROR: failed to establish gRPC connection: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Println("gRPC connection established successfully")
	client := koggerservicerpc.NewKoggerServiceClient(conn)

	fmt.Println("Calling GetLogs...")
	res, err := client.GetLogs(ctx, &koggerservicerpc.LogsRequest{
		Namespace: "default",
	})
	if err != nil {
		fmt.Printf("ERROR: failed to get logs: %v\n", err)
		return
	}
	if len(res.Pods) == 0 {
		fmt.Println("No pods found")
		return
	}
	for _, pod := range res.Pods {
		fmt.Printf("Pod Name: %s, Namespace: %s\n", pod.Name, pod.Namespace)
		fmt.Printf("Log: %s\n", pod.Logs)
	}
	fmt.Println("Cron job completed successfully")
}

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
