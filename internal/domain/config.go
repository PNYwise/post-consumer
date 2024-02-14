package domain

type ExtConf struct {
	Kafka *Kafka `json:"kafka"`
}
type Kafka struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Topic     string `json:"topic"`
	Partition int32  `json:"partition"`
}
