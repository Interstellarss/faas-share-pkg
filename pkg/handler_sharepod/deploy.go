package handlersharepod

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	//"github.com/Interstellarss/faas-share-pkg/pkg/handlersharepod"
	"github.com/Interstellarss/faas-share-pkg/pkg/sharepod"
	k8s "github.com/openfaas/faas-netes/pkg/k8s"
	"github.com/openfaas/faas-provider/types"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const initialReplicasCount = 1

func MakeDeployHandler(funciotnNamespace string, factory k8s.FunctionFactory) http.HandlerFunc {
	secret := k8s.NewSecretsClient(factory.Client)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Body != nil {
			defer r.Body.Close()
		}

		body, _ := ioutil.ReadAll(r.Body)

		request := types.FunctionDeployment{}
		err := json.Unmarshal(body, &request)

		if err != nil {
			wrappedErr := fmt.Errorf("failed ot unarshal request: %s", err.Error())
			http.Error(w, wrappedErr.Error(), http.StatusBadRequest)
			return
		}

		if err := ValidateDeployRequest(&request); err != nil {
			wrappedErr := fmt.Errorf("validation failed: %s", err.Error())
			http.Error(w, wrappedErr.Error(), http.StatusBadRequest)
			return
		}

		namespace := funciotnNamespace
		if len(request.Namespace) > 0 {
			namespace = request.Namespace
		}
		existingSecrets, err := secret.GetSecrets(namespace, request.Secrets)
		if err != nil {
			wrappedErr := fmt.Errorf("undable to fetch secrests: %s", err.Error())
			http.Error(w, wrappedErr.Error(), http.StatusBadRequest)
			return
		}

		deploymentSpec, specErr := makeDeploymentSpec(request, existingSecrets, factory)

		var profileList []k8s.Profile
		if request.Annotations != nil {
			profileNamespace := factory.Config.ProfilesNamespace
			profileList, err = factory.GetProfiles(ctx, profileNamespace, *request.Annotations)
			if err != nil {
				wrappedErr := fmt.Errorf("failed create Deployment spec: %s", err.Error())
				log.Println(wrappedErr)
				http.Error(w, wrappedErr.Error(), http.StatusBadRequest)
				return
			}
		}

		for _, profile := range profileList {
			factory.ApplyProfile(profile, deploymentSpec)
		}

		if specErr != nil {
			wrappedErr := fmt.Errorf("failed create Deployment spc: %s", specErr.Error())
			log.Println(wrappedErr)
			http.Error(w, wrappedErr.Error(), http.StatusBadRequest)
			return
		}

		deploy := factory.Client.AppsV1().Deployments(namespace)

		_, err = deploy.Create(context.TODO(), deploymentSpec, metav1.CreateOptions{})

		if err != nil {
			wrappedErr := fmt.Errorf("unable create Deployment: %s", err.Error())
			log.Println(wrappedErr)
			http.Error(w, wrappedErr.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Deployment created: %s.%s\n", request.Service, namespace)

		service := factory.Client.CoreV1().Services(namespace)
		serviceSpec := makeServiceSpec(request, factory)
		_, err = service.Create(context.TODO(), serviceSpec, metav1.CreateOptions{})

		if err != nil {
			wrappedErr := fmt.Errorf("failed create Service: %s", err.Error())
			log.Println(wrappedErr)
			http.Error(w, wrappedErr.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("Service created: %s.%s\n", request.Service, namespace)
		w.WriteHeader(http.StatusAccepted)

	}
}

func makeDeploymentSpec(request types.FunctionDeployment, existingSecrets map[string]*apiv1.Secret, factory k8s.FunctionFactory) (*appsv1.Deployment, error) {
	envVars := buildEnvVars(&request)

	initialReplicas := int32p(initialReplicasCount)

	labels := map[string]string{
		"faas-share_function": request.Service,
	}

	if request.Labels != nil {
		if min := getMinReplicaCount(*request.Labels); min != nil {
			initialReplicas = min
		}
		for k, v := range *request.Labels {
			labels[k] = v
		}
	}

	nodeselector := createSelector(request.Constraints)

	resource, resourceErr := createResources(request)

	if resourceErr != nil {
		return nil, resourceErr
	}

	var imagePullPolicy apiv1.PullPolicy
	switch factory.Config.ImagePullPolicy {
	case "Never":
		imagePullPolicy = apiv1.PullNever
	case "IfNotPresent":
		imagePullPolicy = apiv1.PullIfNotPresent
	default:
		imagePullPolicy = apiv1.PullAlways
	}

	//todo: here continue
	annotations := buildAnnotations(request)

	probes, err := factory.MakeProbes(request)

	if err != nil {
		return nil, err
	}

	enableServiceLinks := false
	allowPrivilegeEscalation := false

	deploymentSpec := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        request.Service,
			Annotations: annotations,
			Labels: map[string]string{
				"faas-share_function": request.Service,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"faas-share_function": request.Service,
				},
			},
			Replicas: initialReplicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(0),
					},
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(1),
					},
				},
			},
			RevisionHistoryLimit: int32p(10),
			//TODO: does here need to modify for kubeshare
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        request.Service,
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: apiv1.PodSpec{
					//we may not need this part
					NodeSelector: nodeselector,
					Containers: []apiv1.Container{
						{
							Name:  request.Service,
							Image: request.Image,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: factory.Config.RuntimeHTTPPort,
									Protocol:      corev1.ProtocolTCPapiv1.ProtocolTCP,
								},
							},
							Env:             envVars,
							Resources:       *resources,
							ImagePullPolicy: imagePullPolicy,
							LivenessProbe:   probes.Liveness,
							ReadinessProbe:  probes.Liveness,
							SecurityContext: &corev1.SecurityContext{
								ReadOnlyRootFilesystem:   &reques.ReadOnlyRootFilesystem,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					DNSPolicy:     corev1.DNSClusterFirst,

					EnableServiceLinks: &enableServiceLinks,
				},
			},
		},
	}

	factory.ConfigureReadOnlyRootFilesystem(request, deploymentSpec)
	factory.ConfigureContainerUserID(deploymentSpec)

	if err := factory.ConfigureSecrets(request, deploymentSpec, existingSecrets); err != nil {
		return nil, err
	}

	return deploymentSpec, nil
}

func makeServiceSpec(request types.FunctionDeployment, factory k8s.FunctionFactory) *corev1.Service {
	serviceSpec := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Sharepod",
			APIVersion: "kubeshare.nthu/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        request.Service,
			Annotations: buildAnnotations(request),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"faas-share_function": request.Service,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: corev1.ProtocolTCP,
					Port:     factory.Config.RuntimeHTTPPort,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: factory.Config.RuntimeHTTPPort,
					},
				},
			},
		},
	}
	return serviceSpec
}

func buildAnnotations(request types.FunctionDeployment) map[string]string {
	var annotations map[string]string
	if request.Annotations != nil {
		annotations = *request.Annotations
	} else {
		annotations = map[string]string{}
	}

	if _, ok := annotations["prometheus.io.scrape"]; !ok {
		annotations["prometheus.io.scrapte"] = "false"
	}
	return annotations
}

func buildEnvVars(request *types.FunctionDeployment) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

	if len(request.EnvProcess) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k8s.EnvProcessName,
			Value: request.EnvProcess,
		})
	}
	for k, v := range request.EnvVars {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	sort.SliceStable(envVars, func(i, j int) bool {
		return strings.Compare(envVars[i].Name, envVars[j].Name) == -1
	})

	return envVars
}

func createSelector(constraints []string) map[string]string {
	selector := make(map[string]string)

	if len(constraints) > 0 {
		for _, constraint := range constraints {
			parts := strings.Split(constraint, "=")

			if len(parts) == 2 {
				selector[parts[0]] = parts[1]
			}
		}
	}

	return selector
}

//TODO: make it fit for types.FunctionDeployment
func CreateResources(request sharepod.SharepodDeployment) (*sharepod.SharepodRequirements, error) {
	//need to modify here
	/*
		resources := &apiv1.ResourceRequiremtns{
			Limits:  apiv1.ResourceList{},
			Request: apiv1.Resourcelist{},
		}
	*/
	//Instantialize SharepodRequirements
	resources := &sharepod.SharepodRequirements{}

	if request.Resources != nil && len(request.Resources.Memory) > 0 {
		qty, err := resource.ParseQuantity((request.Resources.Memory))

		if err != nil {
			return resources, err
		}
		resources.Memory = qty
	}
	/*
		if request.Requests != nil && len(request.Requests.Memory) > 0 {
			qty, err := resource.ParseQuantity(request.Limits.Memory)
			if err != nil {
				return resources, err
			}
			resources.Limits[apiv1.ResourceMemory] = qty
		}
	*/

	if request.Resources != nil && len(request.Resources.GPULimit) > 0 {
		qty, err := resource.ParseQuantity((request.Resources.GPULimit))
		if err != nil {
			return resources, err
		}
		//todo apiv1 does not have GPU resource
		resources.GPULimit = qty
	}

	if request.Resources != nil && len(request.Resources.GPURequest) > 0 {
		qty, err := resource.ParseQuantity((request.Resources.GPURequest))
		if err != nil {
			return resources, err
		}

		resources.GPURequest = qty
	}

	return resources, nil
}

func getMinReplicaCount(labels map[string]string) *int32 {
	if value, exists := labels["com.openfaas.scale.min"]; exists {
		minReplicas, err := strconv.Atoi(value)
		if err == nil && minReplicas > 0 {
			return int32p(int32(minReplicas))
		}

		log.Println(err)
	}

	return nil
}

func int32p(i int32) *int32 {
	return &i
}
