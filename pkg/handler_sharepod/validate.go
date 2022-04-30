package handlersharepod

import (
	"fmt"
	"regexp"

	"github.com/Interstellarss/faas-share-pkg/pkg/sharepod"
	"github.com/openfaas/faas-provider/types"
)

var validDNS = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func validateDeployRequest(request *sharepod.SharepodDeployment) error {
	//ToDO: service may need to cahnge here?
	matched := validDNS.MatchString(request.Service)
	if matched {
		return nil
	}

	return fmt.Errorf("(%s) must be a valid DNS entry for service name", request.Service)
}

func ValidateDeployRequest(request *types.FunctionDeployment) error {
	//ToDO: service may need to cahnge here?
	matched := validDNS.MatchString(request.Service)
	if matched {
		return nil
	}

	return fmt.Errorf("(%s) must be a valid DNS entry for service name", request.Service)
}
