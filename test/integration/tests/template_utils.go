package tests

import (
	"fmt"
	"strings"

	"github.com/tidwall/sjson"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// applyResourceFromTemplate applies a Kubernetes resource from a template file
// Using oc process to substitute parameters and then apply the result
func applyResourceFromTemplate(tc *TestClient, args ...string) error {
	// Extract the template file from args (look for -f flag)
	var templateFile string
	var processArgs []string
	var applyArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "-f" && i+1 < len(args) {
			templateFile = args[i+1]
			i++ // Skip the next arg as we consumed it
		} else if strings.HasPrefix(args[i], "-p") || (args[i] == "-p" && i+1 < len(args)) {
			// Parameter substitution args go to oc process
			if args[i] == "-p" && i+1 < len(args) {
				processArgs = append(processArgs, args[i], args[i+1])
				i++ // Skip the next arg
			} else {
				processArgs = append(processArgs, args[i])
			}
		} else {
			// Other args go to oc apply
			applyArgs = append(applyArgs, args[i])
		}
	}

	if templateFile == "" {
		return fmt.Errorf("no template file specified with -f flag")
	}

	// Process the template with parameter substitution
	processCmd := tc.AsAdmin().WithoutNamespace().Run("process").Args(append([]string{"-f", templateFile}, processArgs...)...)
	processedYAML, err := processCmd.Output()
	if err != nil {
		e2e.Logf("Failed to process template %s: %v", templateFile, err)
		return err
	}

	// Apply the processed template
	applyCmd := tc.AsAdmin().WithoutNamespace().Run("apply").Args(applyArgs...)
	applyCmd.inputStdin = processedYAML
	err = applyCmd.Execute()
	if err != nil {
		e2e.Logf("Failed to apply processed template: %v", err)
		return err
	}

	e2e.Logf("Successfully applied template %s", templateFile)
	return nil
}

// applyResourceFromTemplateWithOutput applies a template and returns the output
func applyResourceFromTemplateWithOutput(tc *TestClient, args ...string) (string, error) {
	// Extract the template file from args
	var templateFile string
	var processArgs []string
	var applyArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "-f" && i+1 < len(args) {
			templateFile = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "-p") || (args[i] == "-p" && i+1 < len(args)) {
			if args[i] == "-p" && i+1 < len(args) {
				processArgs = append(processArgs, args[i], args[i+1])
				i++
			} else {
				processArgs = append(processArgs, args[i])
			}
		} else {
			applyArgs = append(applyArgs, args[i])
		}
	}

	if templateFile == "" {
		return "", fmt.Errorf("no template file specified with -f flag")
	}

	// Process the template
	processedYAML, err := tc.AsAdmin().WithoutNamespace().Run("process").Args(append([]string{"-f", templateFile}, processArgs...)...).Output()
	if err != nil {
		return "", err
	}

	// Apply the processed template
	applyCmd := tc.AsAdmin().WithoutNamespace().Run("apply").Args(applyArgs...)
	applyCmd.inputStdin = processedYAML
	output, err := applyCmd.Output()
	return output, err
}

// applyResourceFromTemplateWithExtraParametersAsAdmin applies a template with extra JSON parameters as admin
func applyResourceFromTemplateWithExtraParametersAsAdmin(tc *TestClient, extraParameters map[string]interface{}, args ...string) error {
	// Extract template file and parameters
	var templateFile string
	var processArgs []string
	var applyArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "-f" && i+1 < len(args) {
			templateFile = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "-p") || (args[i] == "-p" && i+1 < len(args)) {
			if args[i] == "-p" && i+1 < len(args) {
				processArgs = append(processArgs, args[i], args[i+1])
				i++
			} else {
				processArgs = append(processArgs, args[i])
			}
		} else {
			applyArgs = append(applyArgs, args[i])
		}
	}

	// Process the template
	processedYAML, err := tc.AsAdmin().WithoutNamespace().Run("process").Args(append([]string{"-f", templateFile}, processArgs...)...).Output()
	if err != nil {
		return err
	}

	// Modify the processed JSON with extra parameters
	if jsonPath, ok := extraParameters["jsonPath"].(string); ok {
		delete(extraParameters, "jsonPath")
		for key, value := range extraParameters {
			fullPath := jsonPath + key
			processedYAML, err = sjson.Set(processedYAML, fullPath, value)
			if err != nil {
				e2e.Logf("Failed to set JSON path %s: %v", fullPath, err)
				return err
			}
		}
	}

	// Apply the modified template
	applyCmd := tc.AsAdmin().WithoutNamespace().Run("apply").Args(applyArgs...)
	applyCmd.inputStdin = processedYAML
	return applyCmd.Execute()
}

// applyResourceFromTemplateWithMultiExtraParameters applies a template with multiple JSON modifications
func applyResourceFromTemplateWithMultiExtraParameters(tc *TestClient, jsonPathsAndActions []map[string]string, multiExtraParameters []map[string]interface{}, args ...string) (string, error) {
	// Extract template file and parameters
	var templateFile string
	var processArgs []string
	var applyArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "-f" && i+1 < len(args) {
			templateFile = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "-p") || (args[i] == "-p" && i+1 < len(args)) {
			if args[i] == "-p" && i+1 < len(args) {
				processArgs = append(processArgs, args[i], args[i+1])
				i++
			} else {
				processArgs = append(processArgs, args[i])
			}
		} else {
			applyArgs = append(applyArgs, args[i])
		}
	}

	// Process the template
	processedYAML, err := tc.AsAdmin().WithoutNamespace().Run("process").Args(append([]string{"-f", templateFile}, processArgs...)...).Output()
	if err != nil {
		return "", err
	}

	// Apply multiple JSON path modifications
	for i, pathAndAction := range jsonPathsAndActions {
		for path, action := range pathAndAction {
			if action == "delete" {
				processedYAML, err = sjson.Delete(processedYAML, path)
				if err != nil {
					return "", fmt.Errorf("failed to delete JSON path %s: %v", path, err)
				}
			} else if action == "set" && i < len(multiExtraParameters) {
				// Set the entire object at this path
				for key, value := range multiExtraParameters[i] {
					fullPath := path + key
					processedYAML, err = sjson.Set(processedYAML, fullPath, value)
					if err != nil {
						return "", fmt.Errorf("failed to set JSON path %s: %v", fullPath, err)
					}
				}
			}
		}
	}

	// Apply the modified template
	applyCmd := tc.AsAdmin().WithoutNamespace().Run("apply").Args(applyArgs...)
	applyCmd.inputStdin = processedYAML
	return applyCmd.Output()
}

// applyResourceFromTemplateAsAdmin applies a template as admin
func applyResourceFromTemplateAsAdmin(tc *TestClient, args ...string) error {
	return applyResourceFromTemplate(tc.AsAdmin(), args...)
}

// applyResourceFromTemplateDeleteParametersAsAdmin applies a template and deletes specified JSON paths
func applyResourceFromTemplateDeleteParametersAsAdmin(tc *TestClient, deletePaths []string, args ...string) error {
	// Extract template file and parameters
	var templateFile string
	var processArgs []string
	var applyArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "-f" && i+1 < len(args) {
			templateFile = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "-p") || (args[i] == "-p" && i+1 < len(args)) {
			if args[i] == "-p" && i+1 < len(args) {
				processArgs = append(processArgs, args[i], args[i+1])
				i++
			} else {
				processArgs = append(processArgs, args[i])
			}
		} else {
			applyArgs = append(applyArgs, args[i])
		}
	}

	// Process the template
	processedYAML, err := tc.AsAdmin().WithoutNamespace().Run("process").Args(append([]string{"-f", templateFile}, processArgs...)...).Output()
	if err != nil {
		return err
	}

	// Delete specified JSON paths
	for _, path := range deletePaths {
		processedYAML, err = sjson.Delete(processedYAML, path)
		if err != nil {
			e2e.Logf("Failed to delete JSON path %s: %v", path, err)
			return err
		}
	}

	// Apply the modified template
	applyCmd := tc.AsAdmin().WithoutNamespace().Run("apply").Args(applyArgs...)
	applyCmd.inputStdin = processedYAML
	return applyCmd.Execute()
}
