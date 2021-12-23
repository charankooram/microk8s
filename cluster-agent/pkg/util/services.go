package util

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Restart a MicroK8s service, handling the case where the MicroK8s cluster is running Kubelite.
func Restart(ctx context.Context, serviceName string) error {
	switch serviceName {
	case "kube-apiserver", "kube-proxy", "kube-scheduler", "kube-controller-manager":
		// drop kube- prefix
		serviceName = serviceName[5:]
	}
	if HasKubeliteLock() {
		switch serviceName {
		case "apiserver", "proxy", "kubelet", "scheduler", "controller-manager":
			serviceName = "kubelite"
		}
	}
	return RunCommand(ctx, []string{"snapctl", "restart", fmt.Sprintf("microk8s.daemon-%s", serviceName)})
}

// GetServiceArgument retrieves the value of a specific argument from the $SNAP_DATA/args/$service file.
// The argument name should include preceeding dashes (e.g. "--secure-port").
// If any errors occur, or the argument is not present, an empty string is returned.
func GetServiceArgument(serviceName string, argument string) string {
	arguments, err := ReadFile(SnapDataPath("args", serviceName))
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(arguments, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, argument) {
			continue
		}
		// parse "--argument value" and "--argument=value" variants
		line = line[strings.LastIndex(line, " ")+1:]
		line = line[strings.LastIndex(line, "=")+1:]
		return line
	}
	return ""
}

// UpdateServiceArguments updates the arguments file for a service.
// update is a map of key-value pairs. It will replace the argument with the new value (or just append).
// delete is a list of arguments to remove completely. The argument is removed if present.
func UpdateServiceArguments(serviceName string, update map[string]string, delete []string) error {
	argumentsFile := SnapDataPath("args", serviceName)
	arguments, err := ReadFile(argumentsFile)
	if err != nil {
		return fmt.Errorf("failed to read arguments of service %s: %w", serviceName, err)
	}

	if update == nil {
		update = map[string]string{}
	}
	if delete == nil {
		delete = []string{}
	}

	deleteMap := make(map[string]struct{}, len(delete))
	for _, k := range delete {
		deleteMap[k] = struct{}{}
	}

	newArguments := make([]string, 0, len(arguments))
	for _, line := range strings.Split(arguments, "\n") {
		line = strings.TrimSpace(line)
		// ignore empty lines
		if line == "" {
			continue
		}
		// handle "--argument value" and "--argument=value" variants
		key := strings.SplitN(line, " ", 2)[0]
		key = strings.SplitN(key, "=", 2)[0]
		if newValue, ok := update[key]; ok {
			// update argument with new value
			newArguments = append(newArguments, fmt.Sprintf("%s=%s", key, newValue))
		} else if _, ok := deleteMap[key]; ok {
			// remove argument
			continue
		} else {
			// no change
			newArguments = append(newArguments, line)
		}
	}

	if err := os.WriteFile(argumentsFile, []byte((strings.Join(newArguments, "\n") + "\n")), 0660); err != nil {
		return fmt.Errorf("failed to update arguments for service %s: %q", serviceName, err)
	}
	return nil
}