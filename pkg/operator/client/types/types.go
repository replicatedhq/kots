package types

// TODO: can this type be removed? all fields exist in HelmChart
// OrderedDir represents a kots.io/v1beta1 HelmChart deployment
type OrderedDir struct {
	Name         string
	Weight       int64
	ChartName    string
	ChartVersion string
	ReleaseName  string
	Namespace    string
	UpgradeFlags []string
}

func (o *OrderedDir) GetAPIVersion() string {
	return "kots.io/v1beta1"
}

func (o *OrderedDir) GetChartName() string {
	return o.ChartName
}

func (o *OrderedDir) GetChartVersion() string {
	return o.ChartVersion
}

func (o *OrderedDir) GetReleaseName() string {
	return o.ReleaseName
}

func (o *OrderedDir) GetDirName() string {
	if o.ReleaseName != "" {
		return o.ReleaseName
	}
	return o.ChartName
}

func (o *OrderedDir) GetNamespace() string {
	return o.Namespace
}

func (o *OrderedDir) GetUpgradeFlags() []string {
	return o.UpgradeFlags
}

func (o *OrderedDir) GetWeight() int64 {
	return o.Weight
}
