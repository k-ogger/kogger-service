package kogger

import (
	"bufio"
	"context"
	"fmt"
	"sync"
	"time"

	grpctoken "github.com/ZolaraProject/library/grpctoken"
	logger "github.com/ZolaraProject/library/logger"
	. "github.com/k-ogger/kogger-service/koggerservicerpc"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	NamespaceLogsPathInject = "/api/logs/{%s}"
)

func (*server) GetNamespaces(ctx context.Context, req *Void) (*Namespaces, error) {
	grpcToken := grpctoken.GetToken(ctx)

	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Err(grpcToken, "Failed to retrieve in-cluster Kubernetes config: %s", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		logger.Err(grpcToken, "Failed to initialize Kubernetes client: %s", err)
		return nil, err
	}

	namespacesList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Err(grpcToken, "Failed to list namespaces: %s", err)
		return nil, err
	}

	namespaces := []*Namespace{}
	for _, ns := range namespacesList.Items {
		namespaces = append(namespaces, &Namespace{
			Name: ns.Name,
			Path: fmt.Sprintf(NamespaceLogsPathInject, ns.Name),
		})
	}

	logger.Debug(grpcToken, "Returning %d namespaces", len(namespaces))
	return &Namespaces{
		Namespaces: namespaces,
	}, nil
}

func (*server) GetLogs(ctx context.Context, req *LogsRequest) (*Pods, error) {
	grpcToken := grpctoken.GetToken(ctx)

	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Err(grpcToken, "Failed to retrieve in-cluster Kubernetes config: %s", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		logger.Err(grpcToken, "Failed to initialize Kubernetes client: %s", err)
		return nil, err
	}

	var podsToProcess []v1.Pod
	if len(req.GetNamespace()) != 0 {
		if len(req.GetPod()) != 0 {
			logger.Debug(grpcToken, "Fetching logs for pod %s in namespace %s", req.GetPod(), req.GetNamespace())

			pod, err := clientset.CoreV1().Pods(req.GetNamespace()).Get(ctx, req.GetPod(), metav1.GetOptions{})
			if err != nil {
				logger.Err(grpcToken, "Failed to get pod %s in namespace %s: %s", req.GetPod(), req.GetNamespace(), err)
				return nil, err
			}
			podsToProcess = []v1.Pod{*pod}
		} else {
			logger.Debug(grpcToken, "Fetching logs for all pods in namespace %s", req.GetNamespace())
			allpods, err := clientset.CoreV1().Pods(req.GetNamespace()).List(ctx, metav1.ListOptions{})
			if err != nil {
				logger.Err(grpcToken, "Failed to list pods in namespace %s: %s", req.GetNamespace(), err)
				return nil, err
			}
			podsToProcess = allpods.Items
		}
	} else {
		allpods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Err(grpcToken, "Failed to list pods: %s", err)
			return nil, err
		}
		podsToProcess = allpods.Items
	}

	podLogsChan := make(chan *Pod, len(podsToProcess))
	wg := &sync.WaitGroup{}

	for _, pod := range podsToProcess {
		if pod.Status.Phase != v1.PodRunning {
			logger.Debug(grpcToken, "Skipping pod %s in namespace %s - status: %s", pod.Name, pod.Namespace, pod.Status.Phase)
			continue
		}

		wg.Add(1)
		logger.Debug(grpcToken, "Pod found: %s in namespace %s", pod.Name, pod.Namespace)
		go func(pod v1.Pod) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{})
			podLogs, err := req.Stream(ctx)
			if err != nil {
				logger.Warn(grpcToken, "Failed to get logs for pod %s in namespace %s: %s", pod.Name, pod.Namespace, err)
				return
			}
			defer podLogs.Close()

			logs := ""
			scanner := bufio.NewScanner(podLogs)
			for scanner.Scan() {
				logs += scanner.Text() + "\n"
			}

			if err := scanner.Err(); err != nil {
				logger.Warn(grpcToken, "Error reading logs for pod %s in namespace %s: %s", pod.Name, pod.Namespace, err)
				return
			}

			podLogsChan <- &Pod{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    string(pod.Status.Phase),
				NodeName:  pod.Spec.NodeName,
				Logs:      logs,
			}
		}(pod)
	}
	wg.Wait()

	close(podLogsChan)

	pods := &Pods{}
	logger.Debug(grpcToken, "########## Pod Logs ##########")
	for podLog := range podLogsChan {
		if podLog != nil {
			pods.Pods = append(pods.Pods, podLog)
			logger.Debug(grpcToken, "namespace: %s, pod: %s, status: %s", podLog.Namespace, podLog.Name, podLog.Status)
			logger.Debug(grpcToken, "Logs: %s", podLog.Logs)
			logger.Debug(grpcToken, "")
		}
	}
	logger.Debug(grpcToken, "Returning %d pods with logs", len(pods.Pods))
	if len(pods.Pods) == 0 {
		logger.Debug(grpcToken, "No pods found with logs")
		return &Pods{}, nil
	}

	return pods, nil
}
