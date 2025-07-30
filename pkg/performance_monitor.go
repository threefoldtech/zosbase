package pkg

//go:generate zbusc -module node -version 0.0.1 -name performance-monitor -package stubs github.com/threefoldtech/zosbase/pkg+PerformanceMonitor stubs/performance_monitor_stub.go

type PerformanceMonitor interface {
	GetAllTaskResult() (AllTaskResult, error)
	GetIperfTaskResult() (IperfTaskResult, error)
	GetHealthTaskResult() (HealthTaskResult, error)
	GetPublicIpTaskResult() (PublicIpTaskResult, error)
	GetCpuBenchTaskResult() (CpuBenchTaskResult, error)
	// Deprecated
	Get(taskName string) (TaskResult, error)
	GetAll() ([]TaskResult, error)
}

// TaskResult the result test schema
type TaskResult struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Timestamp   uint64      `json:"timestamp"`
	Result      interface{} `json:"result"`
}

// CPUUtilizationPercent represents CPU utilization percentages
type CPUUtilizationPercent struct {
	Host                float64 `json:"host_total"`
	HostUser            float64 `json:"host_user"`
	HostSystem          float64 `json:"host_system"`
	RemoteTotal         float64 `json:"remote_total"`
	RemoteUser          float64 `json:"remote_user"`
	RemoteSystem        float64 `json:"remote_system"`
	RemoteTotalMessage  float64 `json:"remote_total_message"`
	RemoteUserMessage   float64 `json:"remote_user_message"`
	RemoteSystemMessage float64 `json:"remote_system_message"`
}

// IperfResult for iperf test results
type IperfResult struct {
	UploadSpeed   float64               `json:"upload_speed"`   // in bit/sec
	DownloadSpeed float64               `json:"download_speed"` // in bit/sec
	NodeID        uint32                `json:"node_id"`
	NodeIpv4      string                `json:"node_ip"`
	TestType      string                `json:"test_type"`
	Error         string                `json:"error"`
	CpuReport     CPUUtilizationPercent `json:"cpu_report"`
}

type TaskResultBase struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Timestamp   uint64 `json:"timestamp"`
}

// IperfTaskResult represents the result of an iperf task
type IperfTaskResult struct {
	TaskResultBase
	Result []IperfResult `json:"result"`
}

// HealthReport represents a health check report
type HealthReport struct {
	TestName string   `json:"test_name"`
	Errors   []string `json:"errors"`
}

// HealthTaskResult represents the result of a health check task
type HealthTaskResult struct {
	TaskResultBase
	Result HealthReport `json:"result"`
}

// Report represents a public IP validation report
type Report struct {
	Ip     string `json:"ip"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

// PublicIpTaskResult represents the result of a public IP validation task
type PublicIpTaskResult struct {
	TaskResultBase
	Result []Report `json:"result"`
}

// CPUBenchmarkResult holds CPU benchmark results
type CPUBenchmarkResult struct {
	SingleThreaded float64 `json:"single"`
	MultiThreaded  float64 `json:"multi"`
	Threads        int     `json:"threads"`
	Workloads      int     `json:"workloads"`
}

// CpuBenchTaskResult represents the result of a CPU benchmark task
type CpuBenchTaskResult struct {
	TaskResultBase
	Result CPUBenchmarkResult `json:"result"`
}

// AllTaskResult represents all task results combined
type AllTaskResult struct {
	CpuBenchmark CpuBenchTaskResult `json:"cpu_benchmark"`
	HealthCheck  HealthTaskResult   `json:"health_check"`
	Iperf        IperfTaskResult    `json:"iperf"`
	PublicIp     PublicIpTaskResult `json:"public_ip"`
}
