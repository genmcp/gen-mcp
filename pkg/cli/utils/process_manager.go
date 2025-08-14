package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var manager *ProcessManager

func init() {
	filePath, err := GetCacheDir()
	if err != nil {
		panic(err)
	}

	// ensure that the processes file exists
	filePath = filepath.Join(filePath, "processes")
	_, err = os.Stat(filePath)
	if err != nil {
		f, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}
		f.WriteString("{}")
		f.Close()
	}

	manager = &ProcessManager{
		filePath: filePath,
	}
}

type ProcessManager struct {
	fileMux  sync.Mutex
	filePath string
}

func GetProcessManager() *ProcessManager {
	return manager
}

type processes map[string]int

func (pm *ProcessManager) GetProcessId(name string) (int, error) {
	pm.fileMux.Lock()
	defer pm.fileMux.Unlock()

	bytes, err := os.ReadFile(pm.filePath)
	if err != nil {
		return -1, fmt.Errorf("failed to read %s, unable to find pid for genmcp instance: %w", pm.filePath, err)
	}

	processes := processes{}
	err = json.Unmarshal(bytes, &processes)
	if err != nil {
		return -1, fmt.Errorf("failed to deserialize the contents of %s, unable to find pid for genmcp instance: %w", pm.filePath, err)
	}

	pid, ok := processes[name]
	if !ok {
		return -1, fmt.Errorf("no matching pid for genmcp instance")
	}

	return pid, nil
}

func (pm *ProcessManager) SaveProcessId(name string, pid int) error {
	pm.fileMux.Lock()
	defer pm.fileMux.Unlock()

	bytes, err := os.ReadFile(pm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s, unable to save pid for genmcp instance: %w", pm.filePath, err)
	}

	processes := processes{}
	err = json.Unmarshal(bytes, &processes)
	if err != nil {
		return fmt.Errorf("failed to deserialize the contents of %s, unable to save pid for genmcp instance: %w", pm.filePath, err)
	}

	processes[name] = pid

	bytes, err = json.Marshal(processes)
	if err != nil {
		return fmt.Errorf("failed to serialize the processes map, unable to save pid for genmcp instance: %w", err)
	}

	err = os.WriteFile(pm.filePath, bytes, 0644)

	return nil
}

func (pm *ProcessManager) DeleteProcessId(name string) error {
	pm.fileMux.Lock()
	defer pm.fileMux.Unlock()

	bytes, err := os.ReadFile(pm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s, unable to delete pid for genmcp instance: %w", pm.filePath, err)
	}

	processes := processes{}
	err = json.Unmarshal(bytes, &processes)
	if err != nil {
		return fmt.Errorf("failed to deserialize the contents of %s, unable to delete pid for genmcp instance: %w", pm.filePath, err)
	}

	delete(processes, name)

	bytes, err = json.Marshal(processes)
	if err != nil {
		return fmt.Errorf("failed to serialize the processes map, unable to delete pid for genmcp instance: %w", err)
	}

	err = os.WriteFile(pm.filePath, bytes, 0644)

	return nil
}
