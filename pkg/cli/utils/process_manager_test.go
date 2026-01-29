package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestProcessManager(t *testing.T) (*ProcessManager, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "process_manager_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	infoFilePath := filepath.Join(tmpDir, "process_info")
	if err := os.WriteFile(infoFilePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create process_info file: %v", err)
	}

	pm := &ProcessManager{
		infoFilePath: infoFilePath,
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return pm, cleanup
}

func TestSaveProcess(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		info      ProcessInfo
		wantErr   bool
		setup     func(*ProcessManager)
	}{
		{
			name: "happy path - save new process",
			key:  "/path/to/mcpfile.yaml",
			info: ProcessInfo{
				PID:              1234,
				Name:             "test-server",
				Version:          "1.0.0",
				Transport:        "streamablehttp",
				Port:             8080,
				ToolCount:        5,
				PromptCount:      2,
				ResourceCount:    3,
				StartedAt:        time.Now(),
				MCPFilePath:      "/path/to/mcpfile.yaml",
				ServerConfigPath: "/path/to/mcpserver.yaml",
			},
			wantErr: false,
		},
		{
			name: "overwrite existing process",
			key:  "/path/to/mcpfile.yaml",
			info: ProcessInfo{
				PID:       5678,
				Name:      "updated-server",
				Version:   "2.0.0",
				Transport: "stdio",
			},
			wantErr: false,
			setup: func(pm *ProcessManager) {
				_ = pm.SaveProcess("/path/to/mcpfile.yaml", ProcessInfo{
					PID:  1234,
					Name: "old-server",
				})
			},
		},
		{
			name: "empty key",
			key:  "",
			info: ProcessInfo{
				PID:  1234,
				Name: "test-server",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, cleanup := setupTestProcessManager(t)
			defer cleanup()

			if tt.setup != nil {
				tt.setup(pm)
			}

			err := pm.SaveProcess(tt.key, tt.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveProcess() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got, err := pm.GetProcess(tt.key)
				if err != nil {
					t.Errorf("GetProcess() after save failed: %v", err)
				}
				if got.PID != tt.info.PID {
					t.Errorf("SaveProcess() PID = %v, want %v", got.PID, tt.info.PID)
				}
				if got.Name != tt.info.Name {
					t.Errorf("SaveProcess() Name = %v, want %v", got.Name, tt.info.Name)
				}
			}
		})
	}
}

func TestGetProcess(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		setup   func(*ProcessManager)
		wantPID int
		wantErr bool
	}{
		{
			name: "happy path - get existing process",
			key:  "/path/to/mcpfile.yaml",
			setup: func(pm *ProcessManager) {
				_ = pm.SaveProcess("/path/to/mcpfile.yaml", ProcessInfo{
					PID:  1234,
					Name: "test-server",
				})
			},
			wantPID: 1234,
			wantErr: false,
		},
		{
			name:    "process not found",
			key:     "/nonexistent/path",
			setup:   nil,
			wantErr: true,
		},
		{
			name: "get correct process among multiple",
			key:  "/path/two",
			setup: func(pm *ProcessManager) {
				_ = pm.SaveProcess("/path/one", ProcessInfo{PID: 1111})
				_ = pm.SaveProcess("/path/two", ProcessInfo{PID: 2222})
				_ = pm.SaveProcess("/path/three", ProcessInfo{PID: 3333})
			},
			wantPID: 2222,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, cleanup := setupTestProcessManager(t)
			defer cleanup()

			if tt.setup != nil {
				tt.setup(pm)
			}

			got, err := pm.GetProcess(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProcess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.PID != tt.wantPID {
				t.Errorf("GetProcess() PID = %v, want %v", got.PID, tt.wantPID)
			}
		})
	}
}

func TestListProcesses(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*ProcessManager)
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty list",
			setup:     nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "single process",
			setup: func(pm *ProcessManager) {
				_ = pm.SaveProcess("/path/one", ProcessInfo{PID: 1234})
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple processes",
			setup: func(pm *ProcessManager) {
				_ = pm.SaveProcess("/path/one", ProcessInfo{PID: 1111})
				_ = pm.SaveProcess("/path/two", ProcessInfo{PID: 2222})
				_ = pm.SaveProcess("/path/three", ProcessInfo{PID: 3333})
			},
			wantCount: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, cleanup := setupTestProcessManager(t)
			defer cleanup()

			if tt.setup != nil {
				tt.setup(pm)
			}

			got, err := pm.ListProcesses()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListProcesses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("ListProcesses() count = %v, want %v", len(got), tt.wantCount)
			}
		})
	}
}

func TestDeleteProcess(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		setup   func(*ProcessManager)
		wantErr bool
	}{
		{
			name: "happy path - delete existing process",
			key:  "/path/to/mcpfile.yaml",
			setup: func(pm *ProcessManager) {
				_ = pm.SaveProcess("/path/to/mcpfile.yaml", ProcessInfo{PID: 1234})
			},
			wantErr: false,
		},
		{
			name:    "delete nonexistent process - no error",
			key:     "/nonexistent/path",
			setup:   nil,
			wantErr: false,
		},
		{
			name: "delete one of multiple processes",
			key:  "/path/two",
			setup: func(pm *ProcessManager) {
				_ = pm.SaveProcess("/path/one", ProcessInfo{PID: 1111})
				_ = pm.SaveProcess("/path/two", ProcessInfo{PID: 2222})
				_ = pm.SaveProcess("/path/three", ProcessInfo{PID: 3333})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, cleanup := setupTestProcessManager(t)
			defer cleanup()

			if tt.setup != nil {
				tt.setup(pm)
			}

			err := pm.DeleteProcess(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteProcess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify deletion
			_, err = pm.GetProcess(tt.key)
			if err == nil {
				t.Errorf("DeleteProcess() process still exists after deletion")
			}
		})
	}
}

func TestIsProcessAlive(t *testing.T) {
	tests := []struct {
		name string
		pid  int
		want bool
	}{
		{
			name: "current process is alive",
			pid:  os.Getpid(),
			want: true,
		},
		{
			name: "nonexistent PID",
			pid:  999999999,
			want: false,
		},
		{
			name: "PID zero",
			pid:  0,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsProcessAlive(tt.pid)
			if got != tt.want {
				t.Errorf("IsProcessAlive(%d) = %v, want %v", tt.pid, got, tt.want)
			}
		})
	}
}
