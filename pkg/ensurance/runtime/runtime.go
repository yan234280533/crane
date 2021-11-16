package runtime

import (
	"fmt"

	"google.golang.org/grpc"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"github.com/gocrane-io/crane/pkg/ensurance/grpcc"
	"github.com/gocrane-io/crane/pkg/utils/clogs"
)

// runtimeEndpointIsSet is true when RuntimeEndpoint is configured
// runtimeEndpoint is CRI server runtime endpoint
func getRuntimeClientConnection(runtimeEndpoint string, runtimeEndpointIsSet bool) (*grpc.ClientConn, error) {
	clogs.Log().Info("GetRuntimeClientConnection")

	if runtimeEndpointIsSet && runtimeEndpoint == "" {
		return nil, fmt.Errorf("runtime-endpoint is not set")
	}

	if !runtimeEndpointIsSet {
		clogs.Log().Info(fmt.Sprintf("Runtime connect using default endpoints: %v. "+"As the default settings are now deprecated, you should set the "+
			"endpoint instead.", defaultRuntimeEndpoints))
		return grpcc.InitGrpcConnection(defaultRuntimeEndpoints)
	}

	return grpcc.InitGrpcConnection([]string{runtimeEndpoint})
}

// imageEndpoint is CRI server image endpoint, default same as runtime endpoint
// imageEndpointIsSet is true when imageEndpoint is configured
func getImageClientConnection(imageEndpoint string, imageEndpointIsSet bool) (*grpc.ClientConn, error) {
	clogs.Log().Info("GetImageClientConnection")

	if imageEndpoint == "" {
		if imageEndpointIsSet && imageEndpoint == "" {
			return nil, fmt.Errorf("image-endpoint is not set")
		}
	}

	if !imageEndpointIsSet {
		clogs.Log().Info(fmt.Sprintf("Image connect using default endpoints: %v. "+"As the default settings are now deprecated, you should set the "+
			"endpoint instead.", defaultRuntimeEndpoints))
		return grpcc.InitGrpcConnection(defaultRuntimeEndpoints)
	}
	return grpcc.InitGrpcConnection([]string{imageEndpoint})
}

//GetRuntimeClient get the runtime client
func GetRuntimeClient(runtimeEndpoint string, runtimeEndpointIsSet bool) (pb.RuntimeServiceClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := getRuntimeClientConnection(runtimeEndpoint, runtimeEndpointIsSet)
	if err != nil {
		return nil, nil, err
	}
	runtimeClient := pb.NewRuntimeServiceClient(conn)
	return runtimeClient, conn, nil
}

//GetImageClient get the runtime client
func GetImageClient(imageEndpoint string, imageEndpointIsSet bool) (pb.ImageServiceClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := getImageClientConnection(imageEndpoint, imageEndpointIsSet)
	if err != nil {
		return nil, nil, err
	}
	imageClient := pb.NewImageServiceClient(conn)
	return imageClient, conn, nil
}
