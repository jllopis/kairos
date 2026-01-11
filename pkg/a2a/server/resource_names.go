package server

import (
	"fmt"
	"strings"
)

const (
	taskPrefix            = "tasks/"
	pushConfigSegmentName = "pushNotificationConfigs"
)

func parseTaskName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("task name is required")
	}
	if strings.HasPrefix(name, taskPrefix) {
		parts := strings.Split(name, "/")
		if len(parts) != 2 || parts[1] == "" {
			return "", fmt.Errorf("invalid task name %q", name)
		}
		return parts[1], nil
	}
	if strings.Contains(name, "/") {
		return "", fmt.Errorf("invalid task name %q", name)
	}
	return name, nil
}

func parsePushConfigName(name string) (string, string, error) {
	if name == "" {
		return "", "", fmt.Errorf("config name is required")
	}
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != "tasks" || parts[2] != pushConfigSegmentName {
		return "", "", fmt.Errorf("invalid config name %q", name)
	}
	if parts[1] == "" || parts[3] == "" {
		return "", "", fmt.Errorf("invalid config name %q", name)
	}
	return parts[1], parts[3], nil
}

func pushConfigResourceName(taskID, configID string) string {
	return fmt.Sprintf("tasks/%s/%s/%s", taskID, pushConfigSegmentName, configID)
}
