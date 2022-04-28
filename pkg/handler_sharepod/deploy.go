package handlersharepod

import (
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

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

		request := sharepod.SharepodDeployment{}
		err := json.Unmarshal(body, &request)

		if err != nil {
			wrappedErr := fmt.Errorf("failed ot unarshal request: %s", err.Error())
			http.Error(w, wrappedErr.Error(), http.StatusBadRequest)
			return
		}

		if err := validateDeployRequest(&request); err != nil {
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

		deploymentSpec, err := secrets.GetSecrets(namespace, request.Secrets)
		if err != nil {
			wrappedErr := fmt.Errorf("unable to fetch secrets: %s", err, Error())
			http.Error(w, wrappedErr.Error(), http.StatusBadRequest)
			return
		}

		var profileList []k8s.Profile
		if request.Annotations != nil {
			profileNamespace :=
		}







		log.Printf("")
		w.WriteHeader(http.StatusAccpeted)

	}
}

func makeDeploymentSpec(request sharepod.SharepodDeployment, existingSecrets map[string]*apiv1.Secret, factory k8s.FunctionFactory) (*appsv1.Deployment, error) {
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

}

func buildEnvVars(request *sharepod.SharepodDeployment) []corev1.EnvVar {
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
