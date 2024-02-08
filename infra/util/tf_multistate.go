package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BackendConfig struct {
	Bucket        string
	Key           string
	Region        string
	Profile       string
	DynamoDBTable string
}

func readBackendConfig(backendTFVarsPath string) (BackendConfig, error) {
	file, err := os.Open(backendTFVarsPath)
	if err != nil {
		return BackendConfig{}, fmt.Errorf("error reading %s: %v", backendTFVarsPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var backendConfig BackendConfig
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			switch key {
			case "bucket":
				backendConfig.Bucket = strings.Trim(value, `"`)
			case "key":
				backendConfig.Key = strings.Trim(value, `"`)
			case "region":
				backendConfig.Region = strings.Trim(value, `"`)
			case "profile":
				backendConfig.Profile = strings.Trim(value, `"`)
			case "dynamodb_table":
				backendConfig.DynamoDBTable = strings.Trim(value, `"`)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return BackendConfig{}, fmt.Errorf("error scanning %s: %v", backendTFVarsPath, err)
	}
	return backendConfig, nil
}

type Feature struct {
	Name      string `json:"name"`
	Dir       string `json:"dir"`
	StateFile string `json:"stateFile"`
}

func readFeatures(envPath string) ([]Feature, error) {
	filePath := filepath.Join(envPath, "features.json")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var features []Feature
	if err := json.NewDecoder(file).Decode(&features); err != nil {
		return nil, err
	}

	return features, nil
}

func executeTerraform(feature Feature, env, stateFilePath string, backendConfig BackendConfig, backendTFVarsPath string) error {
	fmt.Printf("executing tf for feature: %s\n", feature.Name)

	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current directory: %v", err)
	}

	defer func() {
		if err := os.Chdir(currentDir); err != nil {
			log.Printf("error changing dir back to root: %v", err)
		}
	}()

	if err := os.Chdir(feature.Dir); err != nil {
		return fmt.Errorf("dir change failure to %s: %v", feature.Dir, err)
	}

	// Use the stateFile key from the feature directly for the Terraform backend configuration
	initCmd := exec.Command("terraform", "init",
		fmt.Sprintf("-backend-config=bucket=%s", backendConfig.Bucket),
		fmt.Sprintf("-backend-config=key=%s", feature.StateFile), // Use the feature's stateFile key here
		fmt.Sprintf("-backend-config=region=%s", backendConfig.Region),
		fmt.Sprintf("-backend-config=profile=%s", backendConfig.Profile),
		fmt.Sprintf("-backend-config=dynamodb_table=%s", backendConfig.DynamoDBTable),
		"-reconfigure")

	initCmd.Env = append(os.Environ(), fmt.Sprintf("TF_STATE=%s", stateFilePath))
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr

	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("tf backend init failure in %s: %v", feature.Dir, err)
	}

	fmt.Printf("tf backend init successful in %s.\n", feature.Dir)

	planCmd := exec.Command("terraform", "plan", fmt.Sprintf("-var-file=../../environments/%s/vars.tfvars", env))
	planCmd.Env = append(os.Environ(), fmt.Sprintf("TF_STATE=%s", stateFilePath))
	planCmd.Stdout = os.Stdout
	planCmd.Stderr = os.Stderr
	if err := planCmd.Run(); err != nil {
		return fmt.Errorf("error running tf plan for %s: %v", feature.Dir, err)
	}

	fmt.Printf("tf plan run successfully for %s.\n", feature.Dir)

	var applyConfirmation string
	fmt.Printf("Do you want to apply the changes? (yes/no): ")
	fmt.Scanln(&applyConfirmation)

	if applyConfirmation == "yes" {
		applyCmd := exec.Command("terraform", "apply", "-auto-approve", fmt.Sprintf("-var-file=../../environments/%s/vars.tfvars", env))
		applyCmd.Env = append(os.Environ(), fmt.Sprintf("TF_STATE=%s", stateFilePath))
		applyCmd.Stdout = os.Stdout
		applyCmd.Stderr = os.Stderr
		if err := applyCmd.Run(); err != nil {
			return fmt.Errorf("error running tf apply for %s: %v", feature.Dir, err)
		}
		fmt.Printf("tf apply run successfully for %s.\n", feature.Dir)
	} else {
		fmt.Println("Apply cancelled by user.")
	}

	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: tfRunner <environment>")
		os.Exit(1)
	}
	env := os.Args[1]

	envPath := filepath.Join("infra/environments", env)
	backendTFVarsPath := filepath.Join(envPath, "backend.tfvars")

	backendConfig, err := readBackendConfig(backendTFVarsPath)
	if err != nil {
		log.Fatalf("error reading backend configuration: %v", err)
	}

	features, err := readFeatures(envPath)
	if err != nil {
		log.Fatalf("err reading features from config file in %s: %v", env, err)
	}

	fmt.Printf("executing tf for env: %s\n", env)

	for _, feature := range features {
		stateFilePath := feature.StateFile
		if err := executeTerraform(feature, env, stateFilePath, backendConfig, backendTFVarsPath); err != nil {
			log.Println(err)
		}
	}

	fmt.Printf("terraform completed successfully %s.\n", env)
}
