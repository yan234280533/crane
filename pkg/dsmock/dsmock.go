package dsmock

import (
	ensuranceapi "github.com/gocrane-io/api/ensurance/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DataSource struct {
}

func GetDataSourceFromNodeProbes(probe ensuranceapi.NodeQualityProbe) (*DataSource, error) {

	/*var dataType = common.DataProviderPromType
	providerConfig := providers.DataProviderConfig{}
	switch opts.DataSource {
	case string(common.DataProviderQCloudMonitor):
		dataType = common.DataProviderQCloudMonitor
		providerConfig.DataProviderQCloudMonitorConfig = &opts.DataSourceQCloudConfig
	case string(common.DataProviderMockType):
		dataType = common.DataProviderMockType
		providerConfig.DataProviderMockConfig = &opts.DataSourceMockConfig
	default:
		providerConfig.DataProviderPromConfig = &opts.DataSourcePromConfig
	}

	dataSource, err := providers.DataProviderFactory(dataType, providerConfig)
	if err != nil {
		return err
	}*/

	return &DataSource{}, nil
}

func (ds *DataSource) LatestTimeSeries(metricName string, selector *metav1.LabelSelector) (float64, error) {
	return 0.0, nil
}
