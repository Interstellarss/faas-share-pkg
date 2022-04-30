package handlersharepod

import (
	"testing"

	types "github.com/openfaas/faas-provider/types"
)

func Test_buildAnnotations_Empty_In_CreateRequest(t *testing.T) {
	request := types.FunctionDeployment{}

	annotations := buildAnnotations(request)

	if len(annotations) != 1 {
		t.Errorf("want: %d annotations got: %d", 1, len(annotations))
	}

	v, ok := annotations["prometheus.io.scrape"]
	if !ok {
		t.Errorf("missing prometheus.io.scrape key")
	}

	want := "false"

	if v != want {
		t.Errorf("want: %s for annotation prometheus.io.scrape got: %s", want, v)
	}
}
