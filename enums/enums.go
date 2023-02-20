package enums

type DataSourceType string

const (
	DataSourceTypeEmpty    DataSourceType = ""
	DataSourceTypeSnapshot DataSourceType = "Snapshot"
	DataSourceTypeVolume   DataSourceType = "Volume"
)
