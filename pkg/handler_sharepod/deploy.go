package handlersharepod

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	k8s "github.com/openfaas/faas-netes/pkg/k8s"
	types "github.com/openfaas/faas-provider/types"
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
