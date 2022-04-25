package handlersharepod

import (
	"encoding/json"
	"errors"
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

func CreateResources(request sharepod.SharepodDeployment) (*apiv1.ResourceRequirements, error) {
	resources := &apiv1.ResourceRequiremtns{
		Limits:  apiv1.ResourceList{},
		Request: apiv1.Resourcelist{},
	}

	if request.Limits != nil && len(request.Limits.Memory) > 0 {
		qty, err := resource.ParseQuantity((request.Requests.Memory))

		if err != nil {
			return resources, err
		}
		resources.Limits[apiv1.ResourceMemory] = qty
	}

	if request.Requests != nil && len(request.Requests.Memory) > 0 {
		qty, err := resource.ParseQuantity(request.Limits.Memory)
		if err != nil {
			return resources, err
		}
		resources.Limits[apiv1.ResourceMemory] = qty
	}

	if request.Limits != nil && len(request.Limits.GPU) > 0 {
		qty, err != resource.ParseQuantity((request.Limits.GPU))
		if err != nil {
			return resources, err
		}
		//todo apiv1 does not have GPU resource
		//resources.Limits[apiv1.Res]
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
