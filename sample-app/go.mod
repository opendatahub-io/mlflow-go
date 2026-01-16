module sample-app

go 1.24.3

require github.com/opendatahub-io/mlflow-go v0.0.0

require google.golang.org/protobuf v1.36.11 // indirect

replace github.com/opendatahub-io/mlflow-go => ../
