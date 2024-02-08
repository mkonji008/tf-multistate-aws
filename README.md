# tf-multistate-aws

Utility tooling written in go and designed around the need to use terraform with multiple state files for different env's and features.

## running from cli or pipeline

```
aws sso login --profile (profile) ## ensure points to the correct profile for the resources

go build -o tfrunner_multistate /infra/util/tfrunner_multistate.go

./tfrunner_multistate (env, dev, staging, prod & etc)
```

- There should be a tfrunner_features.json file in each environment directory, it's where the name, directory, and statefile are located for each feature.

```
[
  {
    "name": "featureA",
    "dir": "infra/modules/featureA",
    "stateFile": "featureA.tfstate"
  },
  {
    "name": "featureB",
    "dir": "infra/modules/featureB",
    "stateFile": "featureB.tfstate"
  }
]
```

## file structure

```
mono-root/
|
|-- infra/
|   |-- main.tf
|	|-- outputs.tf
|	|-- provider.tf
|	|-- util/
|	|	|-- tfRunner.go
|	|-- modules/
|	|	|-- featureA/
|	|	|	|-- main.tf
|	|	|	|-- variables.tf
|	|	|	|-- outputs.tf
|	|	|-- featureB/
|	|	|	|-- main.tf
|	|	|	|-- variables.tf
|	|	|	|-- outputs.tf
|	|-- environments/
|		|-- dev/
|		|	|-- dev.auto.tfvars
|		|	|-- backend.tfvars
|       |   |-- features.json
|		|	|--
|		|-- staging/
|		|	|-- staging.auto.tfvars
|		|	|-- backend.tfvars
|       |   |-- features.json
|		|-- prod/
|		|	|-- prod.auto.tfvars
|		|	|-- backend.tfvars
|       |   |-- features.json
|-- .github/
|   |-- workflows/
|       |-- dev.yaml
|       |-- staging.yaml
|       |-- prod.yaml
|
```
