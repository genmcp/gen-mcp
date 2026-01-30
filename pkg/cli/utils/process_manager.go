package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var manager *ProcessManager

func init() {
	cacheDir, err := GetCacheDir()
	if err != nil {
		panic(err)
	}

	// ensure that the processes file exists
	filePath := filepath.Join(cacheDir, "processes")
	_, err = os.Stat(filePath)
	if err != nil {
		f, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}
		_, err = f.WriteString("{}")
		if err != nil {
			panic(err)
		}
		err = f.Close()
		if err != nil {
			panic(err)
		}
	}

	// ensure that the process_info file exists
	infoFilePath := filepath.Join(cacheDir, "process_info")
	_, err = os.Stat(infoFilePath)
	if err != nil {
		f, err := os.Create(infoFilePath)
		if err != nil {
			panic(err)
		}
		_, err = f.WriteString("{}")
		if err != nil {
			panic(err)
		}
		err = f.Close()
		if err != nil {
			panic(err)
		}
	}

	manager = &ProcessManager{
		filePath:     filePath,
		infoFilePath: infoFilePath,
	}
}

type ProcessManager struct {
	fileMux      sync.Mutex
	filePath     string
	infoFilePath string
}

// ProcessInfo contains rich metadata about a running MCP server process.
type ProcessInfo struct {
	PID              int       `json:"pid"`
	Name             string    `json:"name"`
	Version          string    `json:"version"`
	Transport        string    `json:"transport"`
	Port             int       `json:"port,omitempty"`
	ToolCount        int       `json:"toolCount"`
	PromptCount      int       `json:"promptCount"`
	ResourceCount    int       `json:"resourceCount"`
	StartedAt        time.Time `json:"startedAt"`
	MCPFilePath      string    `json:"mcpFilePath"`
	ServerConfigPath string    `json:"serverConfigPath"`
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

	return os.WriteFile(pm.filePath, bytes, 0644)
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

	return os.WriteFile(pm.filePath, bytes, 0644)
}

// processInfoMap is the storage format for process info.
type processInfoMap map[string]ProcessInfo

// SaveProcess stores rich metadata about a running MCP server process.
func (pm *ProcessManager) SaveProcess(key string, info ProcessInfo) error {
	pm.fileMux.Lock()
	defer pm.fileMux.Unlock()

	bytes, err := os.ReadFile(pm.infoFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s, unable to save process info: %w", pm.infoFilePath, err)
	}

	infoMap := processInfoMap{}
	err = json.Unmarshal(bytes, &infoMap)
	if err != nil {
		return fmt.Errorf("failed to deserialize the contents of %s, unable to save process info: %w", pm.infoFilePath, err)
	}

	infoMap[key] = info

	bytes, err = json.Marshal(infoMap)
	if err != nil {
		return fmt.Errorf("failed to serialize the process info map: %w", err)
	}

	return os.WriteFile(pm.infoFilePath, bytes, 0644)
}

// GetProcess retrieves rich metadata about a running MCP server process.
func (pm *ProcessManager) GetProcess(key string) (*ProcessInfo, error) {
	pm.fileMux.Lock()
	defer pm.fileMux.Unlock()

	bytes, err := os.ReadFile(pm.infoFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s, unable to get process info: %w", pm.infoFilePath, err)
	}

	infoMap := processInfoMap{}
	err = json.Unmarshal(bytes, &infoMap)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize the contents of %s, unable to get process info: %w", pm.infoFilePath, err)
	}

	info, ok := infoMap[key]
	if !ok {
		return nil, fmt.Errorf("no matching process info for key: %s", key)
	}

	return &info, nil
}

// ListProcesses returns all stored process info entries.
func (pm *ProcessManager) ListProcesses() (map[string]ProcessInfo, error) {
	pm.fileMux.Lock()
	defer pm.fileMux.Unlock()

	bytes, err := os.ReadFile(pm.infoFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s, unable to list processes: %w", pm.infoFilePath, err)
	}

	infoMap := processInfoMap{}
	err = json.Unmarshal(bytes, &infoMap)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize the contents of %s, unable to list processes: %w", pm.infoFilePath, err)
	}

	return infoMap, nil
}

// DeleteProcess removes a process info entry by key.
func (pm *ProcessManager) DeleteProcess(key string) error {
	pm.fileMux.Lock()
	defer pm.fileMux.Unlock()

	bytes, err := os.ReadFile(pm.infoFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s, unable to delete process info: %w", pm.infoFilePath, err)
	}

	infoMap := processInfoMap{}
	err = json.Unmarshal(bytes, &infoMap)
	if err != nil {
		return fmt.Errorf("failed to deserialize the contents of %s, unable to delete process info: %w", pm.infoFilePath, err)
	}

	delete(infoMap, key)

	bytes, err = json.Marshal(infoMap)
	if err != nil {
		return fmt.Errorf("failed to serialize the process info map: %w", err)
	}

	return os.WriteFile(pm.infoFilePath, bytes, 0644)
}

// IsProcessAlive checks if a process with the given PID is still running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, sending signal 0 checks if the process exists without affecting it.
	// If the process exists and we have permission to signal it, err will be nil.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
