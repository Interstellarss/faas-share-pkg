package handlersharepod

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/Interstellarss/faas-share-pkg/pkg/sharepod"
	k8s "github.com/openfaas/faas-netes/pkg/k8s"
	types "github.com/openfaas/faas-provider/types"

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

		request := types.FunctionDeployment{}
		err := json.Unmarshal(body, &request)

		if err != nil {
			wrappedErr := fmt.Errorf("failed ot unarshal request: %s", err.Error())

		}

	}
}

func makeDeploymentSpec(request sharepod.SharepodDeployment, existingSecrets map[string]*apiv1.Secret, factory k8s.FunctionFactory) (*appsv1.Deployment, error) {
	envVars := buildEnvVars(&request)

}

func buildEnvVars(reuquest *sharepod.SharepodDeployment) []corev1.EnvVar {

}

func CreateResources(request sharepod.SharepodDeployment) (sharepod.SharepodRequirements, error) {
	//need to modify here
	/*
		resources := &apiv1.ResourceRequiremtns{
			Limits:  apiv1.ResourceList{},
			Request: apiv1.Resourcelist{},
		}
	*/
	//Instantialize SharepodRequirements
	resources := &sharepod.SharepodRequirements{
		GPULimit:   0.0,
		GPURequest: 0.0,
		Memory:     0,
	}

	if request.Resources != nil && request.Resources.Memory > 0 {
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
			return resource, err
		}
		//todo apiv1 does not have GPU resource
		resources.GPULimit = qty.AsApproximateFloat64()
	}
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
