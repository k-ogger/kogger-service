package kogger

import (
	"context"
	"fmt"
	"io"
	"strings"

	grpctoken "github.com/ZolaraProject/library/grpctoken"
	logger "github.com/ZolaraProject/library/logger"
	. "github.com/k-ogger/kogger-service/koggerservicerpc"
	"google.golang.org/protobuf/types/known/structpb"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (*server) GetNamespaces(ctx context.Context, req *Void) (*Namespaces, error) {
	grpcToken := grpctoken.GetToken(ctx)

	namespacesList, err := Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Err(grpcToken, "Failed to list namespaces: %s", err)
		return nil, err
	}

	namespaces := []*Namespace{}
	for _, ns := range namespacesList.Items {
		namespaces = append(namespaces, &Namespace{
			Name: ns.Name,
		})
	}

	logger.Debug(grpcToken, "Returning %d namespaces", len(namespaces))
	return &Namespaces{
		Namespaces: namespaces,
	}, nil
}

func (*server) ListResources(ctx context.Context, req *ListResourcesRequest) (*ResourcesResponse, error) {
	grpcToken := grpctoken.GetToken(ctx)
	if len(req.GetNamespace()) == 0 {
		logger.Err(grpcToken, "Namespace not specified")
		return nil, fmt.Errorf("Namespace not specified")
	}

	resourcesList := []*ResourcesList{}
	if len(req.GetResourceType()) > 0 {
		logger.Debug(grpcToken, "Listing resources of type %s in namespace %s", req.GetResourceType(), req.GetNamespace())
		resource, err := getResources(ctx, grpcToken, req.GetNamespace(), StringToResourceType(req.GetResourceType()))
		if err != nil {
			logger.Err(grpcToken, "Failed to get resources: %s", err)
			return nil, err
		}

		var rsc []*ResourceInlist
		for _, resource := range resource.Resources {
			if resource == nil {
				logger.Warn(grpcToken, "Received nil resource in ListResources")
				continue
			}
			logger.Debug(grpcToken, "Resource: %+v", resource)

			rsc = append(rsc, &ResourceInlist{
				Name: resource.Name,
			})
		}
		resourcesList = append(resourcesList, &ResourcesList{
			ResourceType: req.GetResourceType(),
			Resources:    rsc,
		},
		)
	} else {
		logger.Debug(grpcToken, "Listing all resources in namespace %s", req.GetNamespace())

		var resourcesMap = make(map[string][]*ResourceInlist)
		pods, err := Clientset.CoreV1().Pods(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		logger.Debug(grpcToken, "Pods list: %+v", pods)
		if err != nil {
			logger.Warn(grpcToken, "Failed to list pods in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(pods.Items) == 0 {
				logger.Debug(grpcToken, "No pods found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_POD)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_POD)] = []*ResourceInlist{}
				}

				for _, pod := range pods.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_POD)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_POD)], &ResourceInlist{
						Name: pod.Name,
					})
				}
			}
		}

		services, err := Clientset.CoreV1().Services(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list services in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(services.Items) == 0 {
				logger.Debug(grpcToken, "No services found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SERVICE)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SERVICE)] = []*ResourceInlist{}
				}

				for _, service := range services.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SERVICE)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SERVICE)], &ResourceInlist{
						Name: service.Name,
					})
				}
			}
		}

		deployments, err := Clientset.AppsV1().Deployments(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list deployments in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(deployments.Items) == 0 {
				logger.Debug(grpcToken, "No services found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_DEPLOYMENT)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_DEPLOYMENT)] = []*ResourceInlist{}
				}

				for _, deployment := range deployments.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_DEPLOYMENT)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_DEPLOYMENT)], &ResourceInlist{
						Name: deployment.Name,
					})
				}
			}
		}

		statefulsets, err := Clientset.AppsV1().StatefulSets(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list statefulsets in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(statefulsets.Items) == 0 {
				logger.Debug(grpcToken, "No statefulsets found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_STATEFULSET)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_STATEFULSET)] = []*ResourceInlist{}
				}

				for _, statefulset := range statefulsets.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_STATEFULSET)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_STATEFULSET)], &ResourceInlist{
						Name: statefulset.Name,
					})
				}
			}
		}

		configmaps, err := Clientset.CoreV1().ConfigMaps(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list configmaps in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(configmaps.Items) == 0 {
				logger.Debug(grpcToken, "No configmaps found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_CONFIGMAP)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_CONFIGMAP)] = []*ResourceInlist{}
				}

				for _, configmap := range configmaps.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_CONFIGMAP)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_CONFIGMAP)], &ResourceInlist{
						Name: configmap.Name,
					})
				}
			}
		}

		secrets, err := Clientset.CoreV1().Secrets(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list secrets in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(secrets.Items) == 0 {
				logger.Debug(grpcToken, "No secrets found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SECRET)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SECRET)] = []*ResourceInlist{}
				}

				for _, secret := range secrets.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SECRET)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SECRET)], &ResourceInlist{
						Name: secret.Name,
					})
				}
			}
		}

		pvcs, err := Clientset.CoreV1().PersistentVolumeClaims(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list persistentvolumeclaims in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(pvcs.Items) == 0 {
				logger.Debug(grpcToken, "No pvcs found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_PERSISTENTVOLUMECLAIM)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_PERSISTENTVOLUMECLAIM)] = []*ResourceInlist{}
				}

				for _, pvc := range pvcs.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_PERSISTENTVOLUMECLAIM)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_PERSISTENTVOLUMECLAIM)], &ResourceInlist{
						Name: pvc.Name,
					})
				}
			}
		}

		cronjobs, err := Clientset.BatchV1().CronJobs(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list cronjobs in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(cronjobs.Items) == 0 {
				logger.Debug(grpcToken, "No cronjobs found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_CRONJOB)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_CRONJOB)] = []*ResourceInlist{}
				}

				for _, cronjob := range cronjobs.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_CRONJOB)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_CRONJOB)], &ResourceInlist{
						Name: cronjob.Name,
					})
				}
			}
		}

		jobs, err := Clientset.BatchV1().Jobs(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list jobs in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(jobs.Items) == 0 {
				logger.Debug(grpcToken, "No jobs found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_JOB)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_JOB)] = []*ResourceInlist{}
				}

				for _, job := range jobs.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_JOB)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_JOB)], &ResourceInlist{
						Name: job.Name,
					})
				}
			}
		}

		replicasets, err := Clientset.AppsV1().ReplicaSets(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list replicasets in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(replicasets.Items) == 0 {
				logger.Debug(grpcToken, "No replicasets found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_REPLICASET)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_REPLICASET)] = []*ResourceInlist{}
				}

				for _, replicaset := range replicasets.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_REPLICASET)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_REPLICASET)], &ResourceInlist{
						Name: replicaset.Name,
					})
				}
			}
		}

		daemonsets, err := Clientset.AppsV1().DaemonSets(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list daemonsets in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(daemonsets.Items) == 0 {
				logger.Debug(grpcToken, "No daemonsets found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_DAEMONSET)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_DAEMONSET)] = []*ResourceInlist{}
				}

				for _, daemonset := range daemonsets.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_DAEMONSET)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_DAEMONSET)], &ResourceInlist{
						Name: daemonset.Name,
					})
				}
			}
		}

		ingresses, err := Clientset.NetworkingV1().Ingresses(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list ingresses in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(ingresses.Items) == 0 {
				logger.Debug(grpcToken, "No ingresses found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_INGRESS)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_INGRESS)] = []*ResourceInlist{}
				}

				for _, ingress := range ingresses.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_INGRESS)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_INGRESS)], &ResourceInlist{
						Name: ingress.Name,
					})
				}
			}
		}

		networkPolicies, err := Clientset.NetworkingV1().NetworkPolicies(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list networkpolicies in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(networkPolicies.Items) == 0 {
				logger.Debug(grpcToken, "No networkPolicies found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_NETWORKPOLICY)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_NETWORKPOLICY)] = []*ResourceInlist{}
				}

				for _, networkPolicy := range networkPolicies.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_NETWORKPOLICY)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_NETWORKPOLICY)], &ResourceInlist{
						Name: networkPolicy.Name,
					})
				}
			}
		}

		serviceAccounts, err := Clientset.CoreV1().ServiceAccounts(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list serviceaccounts in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(serviceAccounts.Items) == 0 {
				logger.Debug(grpcToken, "No serviceAccounts found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SERVICEACCOUNT)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SERVICEACCOUNT)] = []*ResourceInlist{}
				}

				for _, serviceAccount := range serviceAccounts.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SERVICEACCOUNT)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_SERVICEACCOUNT)], &ResourceInlist{
						Name: serviceAccount.Name,
					})
				}
			}
		}

		endpoints, err := Clientset.CoreV1().Endpoints(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list endpoints in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(endpoints.Items) == 0 {
				logger.Debug(grpcToken, "No endpoints found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ENDPOINTS)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ENDPOINTS)] = []*ResourceInlist{}
				}

				for _, endpoint := range endpoints.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ENDPOINTS)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ENDPOINTS)], &ResourceInlist{
						Name: endpoint.Name,
					})
				}
			}
		}

		roles, err := Clientset.RbacV1().Roles(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list roles in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(roles.Items) == 0 {
				logger.Debug(grpcToken, "No roles found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ROLE)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ROLE)] = []*ResourceInlist{}
				}

				for _, role := range roles.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ROLE)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ROLE)], &ResourceInlist{
						Name: role.Name,
					})
				}
			}
		}

		roleBindings, err := Clientset.RbacV1().RoleBindings(req.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Warn(grpcToken, "Failed to list rolebindings in namespace %s: %s", req.GetNamespace(), err)
		} else {
			if len(roleBindings.Items) == 0 {
				logger.Debug(grpcToken, "No roleBindings found in namespace %s", req.GetNamespace())
			} else {
				if _, ok := resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ROLEBINDING)]; !ok {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ROLEBINDING)] = []*ResourceInlist{}
				}

				for _, roleBinding := range roleBindings.Items {
					resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ROLEBINDING)] = append(resourcesMap[ResourceTypeToString(ResourceType_RESOURCE_TYPE_ROLEBINDING)], &ResourceInlist{
						Name: roleBinding.Name,
					})
				}
			}
		}

		for resourceType, resourceInList := range resourcesMap {
			resourcesList = append(resourcesList, &ResourcesList{
				ResourceType: resourceType,
				Resources:    resourceInList,
			})
		}
	}

	logger.Debug(grpcToken, "Returning %d resources in namespace %s", len(resourcesList), req.GetNamespace())
	return &ResourcesResponse{
		Namespace:     req.GetNamespace(),
		ResourcesList: resourcesList,
	}, nil
}

func getResources(ctx context.Context, grpcToken, namespace string, resourceType ResourceType) (*Resources, error) {
	logger.Debug(grpcToken, "Fetching resources of type %s in namespace %s", resourceType, namespace)

	var res []*Resource
	switch resourceType {
	case ResourceType_RESOURCE_TYPE_POD:
		pods, err := Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Err(grpcToken, "Failed to list pods in namespace %s: %s", namespace, err)
			return nil, err
		}
		for _, pod := range pods.Items {
			res = append(res, &Resource{
				Namespace: pod.Namespace,
				Name:      pod.Name,
				Status:    string(pod.Status.Phase),
			})
		}
	case ResourceType_RESOURCE_TYPE_DEPLOYMENT:
		logger.Debug(grpcToken, "Fetching deployments in namespace %s", namespace)
		deployments, err := Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Err(grpcToken, "Failed to list deployments in namespace %s: %s", namespace, err)
			return nil, err
		}

		for _, deployment := range deployments.Items {

			status := "Unknown"
			if len(deployment.Status.Conditions) > 0 {
				status = string(deployment.Status.Conditions[0].Type)
			}

			res = append(res, &Resource{
				Namespace: deployment.Namespace,
				Name:      deployment.Name,
				Status:    status,
			})
		}
	case ResourceType_RESOURCE_TYPE_SERVICE:
		logger.Debug(grpcToken, "Fetching services in namespace %s", namespace)
		services, err := Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Err(grpcToken, "Failed to list services in namespace %s: %s", namespace, err)
			return nil, err
		}
		for _, service := range services.Items {
			res = append(res, &Resource{
				Namespace: service.Namespace,
				Name:      service.Name,
				Status:    "Active",
			})
		}
	default:
		logger.Err(grpcToken, "Unsupported resource type: %s", resourceType)
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	logger.Debug(grpcToken, "Returning %d resources of type %s in namespace %s", len(res), resourceType, namespace)
	return &Resources{
		Resources: res,
	}, nil
}

func (*server) GetResource(ctx context.Context, req *ResourceRequest) (*Resource, error) {
	grpcToken := grpctoken.GetToken(ctx)

	if len(req.GetNamespace()) == 0 || len(req.GetName()) == 0 || req.GetResourceType() == 0 {
		logger.Err(grpcToken, "Namespace, name or resource type not specified")
		return nil, fmt.Errorf("namespace, name or resource type not specified")
	}

	logger.Debug(grpcToken, "Fetching resource %s of type %s in namespace %s", req.GetName(), ResourceTypeToString(req.GetResourceType()), req.GetNamespace())

	resourceType := ResourceTypeToString(req.GetResourceType())
	var resourceInfo *Resource
	switch req.GetResourceType() {
	case ResourceType_RESOURCE_TYPE_POD:
		resource, err := Clientset.CoreV1().Pods(req.GetNamespace()).Get(ctx, req.GetName(), metav1.GetOptions{})
		if err != nil {
			logger.Err(grpcToken, "Failed to get pod %s in namespace %s: %s", req.GetName(), req.GetNamespace(), err)
			return nil, err
		}
		resourceInfo = &Resource{
			Namespace: resource.Namespace,
			Name:      resource.Name,
			Status:    string(resource.Status.Phase),
		}
	case ResourceType_RESOURCE_TYPE_DEPLOYMENT:
		resource, err := Clientset.AppsV1().Deployments(req.GetNamespace()).Get(ctx, req.GetName(), metav1.GetOptions{})
		if err != nil {
			logger.Err(grpcToken, "Failed to get deployment %s in namespace %s: %s", req.GetName(), req.GetNamespace(), err)
			return nil, err
		}

		resourceInfo = analyseDeployment(resource)
	case ResourceType_RESOURCE_TYPE_SERVICE:
		resource, err := Clientset.CoreV1().Services(req.GetNamespace()).Get(ctx, req.GetName(), metav1.GetOptions{})
		if err != nil {
			logger.Err(grpcToken, "Failed to get service %s in namespace %s: %s", req.GetName(), req.GetNamespace(), err)
			return nil, err
		}

		resourceInfo = analyseService(resource)
	default:
		logger.Err(grpcToken, "Unsupported resource type: %s", resourceType)
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return resourceInfo, nil
}

func (*server) GetLogs(ctx context.Context, req *LogsRequest) (*Logs, error) {
	grpcToken := grpctoken.GetToken(ctx)

	if len(req.GetNamespace()) == 0 || len(req.GetPod()) == 0 {
		logger.Err(grpcToken, "Namespace or pod not specified")
		return nil, fmt.Errorf("namespace or pod not specified")
	}

	logger.Debug(grpcToken, "Fetching logs for pod %s in namespace %s", req.GetPod(), req.GetNamespace())

	pod, err := Clientset.CoreV1().Pods(req.GetNamespace()).Get(ctx, req.GetPod(), metav1.GetOptions{})
	if err != nil {
		logger.Err(grpcToken, "Failed to get pod %s in namespace %s: %s", req.GetPod(), req.GetNamespace(), err)
		return nil, err
	}

	logs := []*LogEntry{}

	for _, container := range pod.Spec.Containers {
		logReq := Clientset.CoreV1().Pods(req.GetNamespace()).GetLogs(pod.Name, &v1.PodLogOptions{
			Container:  container.Name,
			Timestamps: true,
		})
		podLogs, err := logReq.Stream(ctx)
		if err != nil {
			logger.Err(grpcToken, "Failed to get logs for pod %s in namespace %s, container %s: %s", req.GetPod(), req.GetNamespace(), container.Name, err)
			continue
		}

		buf := new(strings.Builder)
		if _, err := io.Copy(buf, podLogs); err != nil {
			logger.Err(grpcToken, "Failed to read logs for pod %s in namespace %s, container %s: %s", req.GetPod(), req.GetNamespace(), container.Name, err)
			podLogs.Close()
			continue
		}
		if err := podLogs.Close(); err != nil {
			logger.Err(grpcToken, "Failed to close log stream for pod %s in namespace %s, container %s: %s", req.GetPod(), req.GetNamespace(), container.Name, err)
		}

		logLines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		for _, line := range logLines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			timestamp := ""
			message := line

			if len(line) > 30 && line[4] == '-' && line[7] == '-' && line[10] == 'T' {
				parts := strings.SplitN(line, " ", 2)
				if len(parts) == 2 {
					timestamp = parts[0]
					message = parts[1]
				}
			}

			logs = append(logs, &LogEntry{
				Container: container.Name,
				Timestamp: timestamp,
				Message:   message,
			})
		}
	}

	return &Logs{
		Pod:       req.GetPod(),
		Namespace: req.GetNamespace(),
		Entries:   logs,
	}, nil
}

func analyseDeployment(deployment *appsv1.Deployment) *Resource {
	deploymentFields := &AdjustableFields{
		Fields: make(map[string]*structpb.Value),
	}

	containerList := []*structpb.Value{}
	imageList := []*structpb.Value{}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		containerList = append(containerList, structpb.NewStringValue(container.Name))
		imageList = append(imageList, structpb.NewStringValue(container.Image))
	}

	labelsList := make(map[string]*structpb.Value)
	for key, value := range deployment.Spec.Template.ObjectMeta.Labels {
		labelsList[key] = structpb.NewStringValue(value)
	}

	deploymentFields.Fields["Containers"] = structpb.NewListValue(&structpb.ListValue{Values: containerList})
	deploymentFields.Fields["Images"] = structpb.NewListValue(&structpb.ListValue{Values: imageList})

	if deployment.Spec.Replicas != nil {
		deploymentFields.Fields["Replicas"] = structpb.NewStringValue(fmt.Sprintf("%d", *deployment.Spec.Replicas))
	} else {
		deploymentFields.Fields["Replicas"] = structpb.NewStringValue("0")
	}
	deploymentFields.Fields["Selector"] = structpb.NewStructValue(&structpb.Struct{
		Fields: labelsList,
	})

	status := "Unknown"
	if len(deployment.Status.Conditions) > 0 {
		status = string(deployment.Status.Conditions[0].Type)
	}

	return &Resource{
		Namespace: deployment.Namespace,
		Name:      deployment.Name,
		Status:    status,
		Fields:    deploymentFields,
	}
}

func analyseService(service *v1.Service) *Resource {
	serviceFields := &AdjustableFields{
		Fields: make(map[string]*structpb.Value),
	}

	serviceFields.Fields["ClusterIP"] = structpb.NewStringValue(service.Spec.ClusterIP)

	ports := make(map[string]*structpb.Value)
	for _, port := range service.Spec.Ports {
		var portname string = port.Name
		if portname == "" {
			portname = "port"
		}

		ports[portname] = structpb.NewStringValue(fmt.Sprintf("%d/%s", port.Port, port.Protocol))
	}
	serviceFields.Fields["Ports"] = structpb.NewStructValue(&structpb.Struct{
		Fields: ports,
	})

	selectorFields := make(map[string]*structpb.Value)
	for key, value := range service.Spec.Selector {
		selectorFields[key] = structpb.NewStringValue(value)
	}
	serviceFields.Fields["Selector"] = structpb.NewStructValue(&structpb.Struct{
		Fields: selectorFields,
	})

	return &Resource{
		Namespace: service.Namespace,
		Name:      service.Name,
		Status:    "Active",
		Fields:    serviceFields,
	}
}
