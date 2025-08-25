package metrics

import (
	"fmt"
	"log"
	"runtime"
	"time"

	pb "github.com/mooncorn/nodelink/agent/internal/proto"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Collector collects system metrics and information
type Collector struct{}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{}
}

// GetSystemInfo collects static system information
func (c *Collector) GetSystemInfo() (*pb.SystemInfo, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	// Get network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Warning: failed to get network interfaces: %v", err)
		interfaces = []net.InterfaceStat{} // Use empty slice if error
	}

	networkInterfaces := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Name != "lo" { // Skip loopback interface
			networkInterfaces = append(networkInterfaces, iface.Name)
		}
	}

	// Get memory info for total memory
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Warning: failed to get memory info: %v", err)
		memInfo = &mem.VirtualMemoryStat{Total: 0}
	}

	return &pb.SystemInfo{
		Hostname:          hostInfo.Hostname,
		Platform:          hostInfo.Platform,
		Arch:              runtime.GOARCH,
		OsVersion:         hostInfo.PlatformVersion,
		CpuCount:          int32(runtime.NumCPU()),
		TotalMemory:       int64(memInfo.Total),
		NetworkInterfaces: networkInterfaces,
		KernelVersion:     hostInfo.KernelVersion,
		UptimeSeconds:     int64(hostInfo.Uptime),
	}, nil
}

// GetSystemMetrics collects current system metrics
func (c *Collector) GetSystemMetrics() (*pb.SystemMetrics, error) {
	// CPU usage
	cpuPercents, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	var cpuUsage float64
	if len(cpuPercents) > 0 {
		cpuUsage = cpuPercents[0]
	}

	// Memory metrics
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	memoryMetrics := &pb.MemoryMetrics{
		Total:       int64(memInfo.Total),
		Available:   int64(memInfo.Available),
		Used:        int64(memInfo.Used),
		UsedPercent: memInfo.UsedPercent,
		Free:        int64(memInfo.Free),
		Cached:      int64(memInfo.Cached),
		Buffers:     int64(memInfo.Buffers),
	}

	// Disk metrics
	diskMetrics, err := c.getDiskMetrics()
	if err != nil {
		log.Printf("Warning: failed to get disk metrics: %v", err)
		diskMetrics = []*pb.DiskMetrics{} // Use empty slice if error
	}

	// Network metrics
	networkMetrics, err := c.getNetworkMetrics()
	if err != nil {
		log.Printf("Warning: failed to get network metrics: %v", err)
		networkMetrics = []*pb.NetworkMetrics{} // Use empty slice if error
	}

	// Process metrics (top 10 by CPU usage)
	processMetrics, err := c.getTopProcesses(10)
	if err != nil {
		log.Printf("Warning: failed to get process metrics: %v", err)
		processMetrics = []*pb.ProcessMetrics{} // Use empty slice if error
	}

	// Load average
	loadInfo, err := load.Avg()
	if err != nil {
		log.Printf("Warning: failed to get load average: %v", err)
		loadInfo = &load.AvgStat{Load1: 0, Load5: 0, Load15: 0}
	}

	return &pb.SystemMetrics{
		CpuUsagePercent:   cpuUsage,
		Memory:            memoryMetrics,
		Disks:             diskMetrics,
		NetworkInterfaces: networkMetrics,
		Processes:         processMetrics,
		Timestamp:         time.Now().Unix(),
		LoadAverage_1M:    loadInfo.Load1,
		LoadAverage_5M:    loadInfo.Load5,
		LoadAverage_15M:   loadInfo.Load15,
	}, nil
}

// getDiskMetrics collects disk usage metrics for all mounted filesystems
func (c *Collector) getDiskMetrics() ([]*pb.DiskMetrics, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var diskMetrics []*pb.DiskMetrics
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			log.Printf("Warning: failed to get disk usage for %s: %v", partition.Mountpoint, err)
			continue
		}

		diskMetric := &pb.DiskMetrics{
			Device:      partition.Device,
			Mountpoint:  partition.Mountpoint,
			Filesystem:  partition.Fstype,
			Total:       int64(usage.Total),
			Used:        int64(usage.Used),
			Free:        int64(usage.Free),
			UsedPercent: usage.UsedPercent,
		}
		diskMetrics = append(diskMetrics, diskMetric)
	}

	return diskMetrics, nil
}

// getNetworkMetrics collects network interface statistics
func (c *Collector) getNetworkMetrics() ([]*pb.NetworkMetrics, error) {
	stats, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	var networkMetrics []*pb.NetworkMetrics
	for _, stat := range stats {
		if stat.Name == "lo" { // Skip loopback interface
			continue
		}

		networkMetric := &pb.NetworkMetrics{
			Interface:   stat.Name,
			BytesSent:   int64(stat.BytesSent),
			BytesRecv:   int64(stat.BytesRecv),
			PacketsSent: int64(stat.PacketsSent),
			PacketsRecv: int64(stat.PacketsRecv),
			ErrorsIn:    int64(stat.Errin),
			ErrorsOut:   int64(stat.Errout),
			DropsIn:     int64(stat.Dropin),
			DropsOut:    int64(stat.Dropout),
		}
		networkMetrics = append(networkMetrics, networkMetric)
	}

	return networkMetrics, nil
}

// getTopProcesses gets the top N processes by CPU usage
func (c *Collector) getTopProcesses(limit int) ([]*pb.ProcessMetrics, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	type processInfo struct {
		pid        int32
		name       string
		cpuPercent float64
		memRss     int64
		memVms     int64
		status     string
		createTime int64
		numThreads int32
	}

	var processes []processInfo
	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue // Process might have ended
		}

		name, err := proc.Name()
		if err != nil {
			continue
		}

		cpuPercent, err := proc.CPUPercent()
		if err != nil {
			cpuPercent = 0 // Default to 0 if can't get CPU usage
		}

		memInfo, err := proc.MemoryInfo()
		if err != nil {
			memInfo = &process.MemoryInfoStat{RSS: 0, VMS: 0}
		}

		status, err := proc.Status()
		var statusStr string
		if err != nil {
			statusStr = "unknown"
		} else if len(status) > 0 {
			statusStr = status[0]
		} else {
			statusStr = "unknown"
		}

		createTime, err := proc.CreateTime()
		if err != nil {
			createTime = 0
		}

		numThreads, err := proc.NumThreads()
		if err != nil {
			numThreads = 0
		}

		processes = append(processes, processInfo{
			pid:        pid,
			name:       name,
			cpuPercent: cpuPercent,
			memRss:     int64(memInfo.RSS),
			memVms:     int64(memInfo.VMS),
			status:     statusStr,
			createTime: createTime,
			numThreads: numThreads,
		})
	}

	// Sort by CPU usage (descending) and take top N
	for i := 0; i < len(processes)-1; i++ {
		for j := i + 1; j < len(processes); j++ {
			if processes[i].cpuPercent < processes[j].cpuPercent {
				processes[i], processes[j] = processes[j], processes[i]
			}
		}
	}

	if len(processes) > limit {
		processes = processes[:limit]
	}

	var processMetrics []*pb.ProcessMetrics
	for _, proc := range processes {
		processMetric := &pb.ProcessMetrics{
			Pid:        proc.pid,
			Name:       proc.name,
			CpuPercent: proc.cpuPercent,
			MemoryRss:  proc.memRss,
			MemoryVms:  proc.memVms,
			Status:     proc.status,
			CreateTime: proc.createTime,
			NumThreads: proc.numThreads,
		}
		processMetrics = append(processMetrics, processMetric)
	}

	return processMetrics, nil
}
