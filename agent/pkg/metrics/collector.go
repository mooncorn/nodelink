package metrics

import (
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	pb "github.com/mooncorn/nodelink/proto"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type MetricsCollector struct {
	interval   time.Duration
	stopCh     chan struct{}
	collecting bool
	mu         sync.RWMutex

	// Client for sending metrics
	sendFunc func(*pb.TaskResponse)

	// Track process metrics by task ID
	taskProcesses map[string][]*process.Process
	taskMutex     sync.RWMutex

	// Previous metrics for calculating rates
	prevNetStats  []net.IOCountersStat
	prevDiskStats map[string]disk.IOCountersStat
	prevTimestamp time.Time

	// Current streaming task ID
	streamingTaskID string
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(sendFunc func(*pb.TaskResponse)) *MetricsCollector {
	return &MetricsCollector{
		sendFunc:      sendFunc,
		taskProcesses: make(map[string][]*process.Process),
		prevDiskStats: make(map[string]disk.IOCountersStat),
	}
}

// StartStreaming starts metrics collection at the specified interval
func (mc *MetricsCollector) StartStreaming(agentID, taskID string, interval time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.collecting {
		return
	}

	mc.interval = interval
	mc.streamingTaskID = taskID
	mc.stopCh = make(chan struct{})
	mc.collecting = true

	go mc.collectLoop(agentID)
	log.Printf("Started metrics collection with interval %v for task %s", interval, taskID)
}

// StopStreaming stops metrics collection
func (mc *MetricsCollector) StopStreaming() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.collecting {
		return
	}

	close(mc.stopCh)
	mc.collecting = false
	log.Println("Stopped metrics collection")
}

// UpdateInterval changes the collection interval
func (mc *MetricsCollector) UpdateInterval(interval time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.interval = interval
	log.Printf("Updated metrics collection interval to %v", interval)
}

// IsCollecting returns whether metrics are currently being collected
func (mc *MetricsCollector) IsCollecting() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.collecting
}

// IsStreamingTask returns whether a specific task ID is the current streaming task
func (mc *MetricsCollector) IsStreamingTask(taskID string) bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.collecting && mc.streamingTaskID == taskID
}

// GetStreamingTaskID returns the current streaming task ID
func (mc *MetricsCollector) GetStreamingTaskID() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.streamingTaskID
}

// AddTaskProcess tracks a process for a specific task
func (mc *MetricsCollector) AddTaskProcess(taskID string, pid int32) {
	mc.taskMutex.Lock()
	defer mc.taskMutex.Unlock()

	proc, err := process.NewProcess(pid)
	if err != nil {
		log.Printf("Failed to track process %d for task %s: %v", pid, taskID, err)
		return
	}

	mc.taskProcesses[taskID] = append(mc.taskProcesses[taskID], proc)
}

// RemoveTaskProcess removes tracking for a task's processes
func (mc *MetricsCollector) RemoveTaskProcess(taskID string) {
	mc.taskMutex.Lock()
	defer mc.taskMutex.Unlock()

	delete(mc.taskProcesses, taskID)
}

// collectLoop runs the collection loop
func (mc *MetricsCollector) collectLoop(agentID string) {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if we're still supposed to be collecting
			mc.mu.RLock()
			collecting := mc.collecting
			taskID := mc.streamingTaskID
			mc.mu.RUnlock()

			if !collecting {
				return
			}

			metrics := mc.collectMetrics()
			if metrics != nil {
				mc.sendMetrics(agentID, taskID, metrics)
			}

		case <-mc.stopCh:
			return
		}
	}
}

// sendMetrics sends metrics data via the configured send function
func (mc *MetricsCollector) sendMetrics(agentID, taskID string, metrics *pb.MetricsDataResponse) {
	if mc.sendFunc == nil {
		return
	}

	response := &pb.TaskResponse{
		AgentId:   agentID,
		TaskId:    taskID,
		Status:    pb.TaskResponse_IN_PROGRESS,
		IsFinal:   false,
		Cancelled: false,
		EventType: "metrics",
		Timestamp: time.Now().Unix(),
		Response: &pb.TaskResponse_MetricsResponse{
			MetricsResponse: &pb.MetricsResponse{
				ResponseType: &pb.MetricsResponse_MetricsData{
					MetricsData: metrics,
				},
			},
		},
	}

	mc.sendFunc(response)
}

// collectMetrics collects current system metrics
func (mc *MetricsCollector) collectMetrics() *pb.MetricsDataResponse {
	timestamp := time.Now()

	return &pb.MetricsDataResponse{
		Timestamp:         uint64(timestamp.Unix()),
		Cpu:               mc.collectCpuMetrics(),
		Memory:            mc.collectMemoryMetrics(),
		Disks:             mc.collectDiskMetrics(),
		NetworkInterfaces: mc.collectNetworkMetrics(),
		Processes:         mc.collectProcessMetrics(),
		Load:              mc.collectLoadMetrics(),
	}
}

// collectCpuMetrics collects CPU metrics
func (mc *MetricsCollector) collectCpuMetrics() *pb.CpuMetrics {
	// Overall CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil || len(cpuPercent) == 0 {
		log.Printf("Failed to get CPU percentage: %v", err)
		return &pb.CpuMetrics{}
	}

	// Per-core usage
	corePercent, err := cpu.Percent(0, true)
	if err != nil {
		log.Printf("Failed to get per-core CPU percentage: %v", err)
		corePercent = []float64{}
	}

	// CPU times
	cpuTimes, err := cpu.Times(false)
	if err != nil || len(cpuTimes) == 0 {
		log.Printf("Failed to get CPU times: %v", err)
		return &pb.CpuMetrics{
			UsagePercent: cpuPercent[0],
			CoreUsage:    corePercent,
		}
	}

	total := cpuTimes[0].User + cpuTimes[0].System + cpuTimes[0].Idle + cpuTimes[0].Nice + cpuTimes[0].Iowait + cpuTimes[0].Irq + cpuTimes[0].Softirq + cpuTimes[0].Steal + cpuTimes[0].Guest + cpuTimes[0].GuestNice
	return &pb.CpuMetrics{
		UsagePercent:  cpuPercent[0],
		CoreUsage:     corePercent,
		UserPercent:   (cpuTimes[0].User / total) * 100,
		SystemPercent: (cpuTimes[0].System / total) * 100,
		IdlePercent:   (cpuTimes[0].Idle / total) * 100,
		IowaitPercent: (cpuTimes[0].Iowait / total) * 100,
		// Temperature requires specific hardware access
		TemperatureCelsius: 0,
	}
}

// collectMemoryMetrics collects memory metrics
func (mc *MetricsCollector) collectMemoryMetrics() *pb.MemoryMetrics {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Failed to get memory info: %v", err)
		return &pb.MemoryMetrics{}
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		log.Printf("Failed to get swap info: %v", err)
		swapInfo = &mem.SwapMemoryStat{}
	}

	return &pb.MemoryMetrics{
		TotalBytes:       memInfo.Total,
		UsedBytes:        memInfo.Used,
		AvailableBytes:   memInfo.Available,
		FreeBytes:        memInfo.Free,
		CachedBytes:      memInfo.Cached,
		BuffersBytes:     memInfo.Buffers,
		UsagePercent:     memInfo.UsedPercent,
		SwapTotalBytes:   swapInfo.Total,
		SwapUsedBytes:    swapInfo.Used,
		SwapFreeBytes:    swapInfo.Free,
		SwapUsagePercent: swapInfo.UsedPercent,
	}
}

// collectDiskMetrics collects disk metrics
func (mc *MetricsCollector) collectDiskMetrics() []*pb.DiskMetrics {
	partitions, err := disk.Partitions(false)
	if err != nil {
		log.Printf("Failed to get disk partitions: %v", err)
		return []*pb.DiskMetrics{}
	}

	var diskMetrics []*pb.DiskMetrics
	currentTime := time.Now()

	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		diskMetric := &pb.DiskMetrics{
			Device:         partition.Device,
			MountPoint:     partition.Mountpoint,
			TotalBytes:     usage.Total,
			UsedBytes:      usage.Used,
			AvailableBytes: usage.Free,
			UsagePercent:   usage.UsedPercent,
		}

		// Get I/O stats if available
		ioStats, err := disk.IOCounters()
		if err == nil {
			if stat, exists := ioStats[partition.Device]; exists {
				// Calculate rates if we have previous stats
				if prevStat, hasPrev := mc.prevDiskStats[partition.Device]; hasPrev && !mc.prevTimestamp.IsZero() {
					timeDiff := currentTime.Sub(mc.prevTimestamp).Seconds()
					if timeDiff > 0 {
						diskMetric.ReadBytesPerSec = uint64(float64(stat.ReadBytes-prevStat.ReadBytes) / timeDiff)
						diskMetric.WriteBytesPerSec = uint64(float64(stat.WriteBytes-prevStat.WriteBytes) / timeDiff)
						diskMetric.ReadOpsPerSec = uint64(float64(stat.ReadCount-prevStat.ReadCount) / timeDiff)
						diskMetric.WriteOpsPerSec = uint64(float64(stat.WriteCount-prevStat.WriteCount) / timeDiff)
					}
				}

				// Store current stats for next calculation
				mc.prevDiskStats[partition.Device] = stat
			}
		}

		diskMetrics = append(diskMetrics, diskMetric)
	}

	mc.prevTimestamp = currentTime
	return diskMetrics
}

// collectNetworkMetrics collects network metrics
func (mc *MetricsCollector) collectNetworkMetrics() []*pb.NetworkMetrics {
	netStats, err := net.IOCounters(true)
	if err != nil {
		log.Printf("Failed to get network stats: %v", err)
		return []*pb.NetworkMetrics{}
	}

	var networkMetrics []*pb.NetworkMetrics
	currentTime := time.Now()

	for _, stat := range netStats {
		netMetric := &pb.NetworkMetrics{
			Interface: stat.Name,
			ErrorsIn:  stat.Errin,
			ErrorsOut: stat.Errout,
			DropsIn:   stat.Dropin,
			DropsOut:  stat.Dropout,
		}

		// Calculate rates if we have previous stats
		if len(mc.prevNetStats) > 0 && !mc.prevTimestamp.IsZero() {
			for _, prevStat := range mc.prevNetStats {
				if prevStat.Name == stat.Name {
					timeDiff := currentTime.Sub(mc.prevTimestamp).Seconds()
					if timeDiff > 0 {
						netMetric.BytesSentPerSec = uint64(float64(stat.BytesSent-prevStat.BytesSent) / timeDiff)
						netMetric.BytesRecvPerSec = uint64(float64(stat.BytesRecv-prevStat.BytesRecv) / timeDiff)
						netMetric.PacketsSentPerSec = uint64(float64(stat.PacketsSent-prevStat.PacketsSent) / timeDiff)
						netMetric.PacketsRecvPerSec = uint64(float64(stat.PacketsRecv-prevStat.PacketsRecv) / timeDiff)
					}
					break
				}
			}
		}

		networkMetrics = append(networkMetrics, netMetric)
	}

	// Store current stats for next calculation
	mc.prevNetStats = netStats
	return networkMetrics
}

// collectProcessMetrics collects process metrics
func (mc *MetricsCollector) collectProcessMetrics() *pb.ProcessMetrics {
	processes, err := process.Processes()
	if err != nil {
		log.Printf("Failed to get process list: %v", err)
		return &pb.ProcessMetrics{}
	}

	processMetrics := &pb.ProcessMetrics{
		TotalProcesses: uint32(len(processes)),
	}

	// Count process states
	for _, proc := range processes {
		status, err := proc.Status()
		if err != nil {
			continue
		}

		if len(status) > 0 {
			switch status[0] {
			case "R":
				processMetrics.RunningProcesses++
			case "S":
				processMetrics.SleepingProcesses++
			case "Z":
				processMetrics.ZombieProcesses++
			}
		}
	}

	// Collect task-specific process metrics
	mc.taskMutex.RLock()
	for taskID, taskProcs := range mc.taskProcesses {
		for _, proc := range taskProcs {
			if proc == nil {
				continue
			}

			// Check if process still exists
			if exists, _ := proc.IsRunning(); !exists {
				continue
			}

			cpuPercent, _ := proc.CPUPercent()
			memInfo, _ := proc.MemoryInfo()
			numThreads, _ := proc.NumThreads()
			ioStats, _ := proc.IOCounters()

			taskMetric := &pb.TaskProcessMetrics{
				TaskId:     taskID,
				Pid:        uint32(proc.Pid),
				CpuPercent: cpuPercent,
				Threads:    uint32(numThreads),
			}

			if memInfo != nil {
				taskMetric.MemoryBytes = memInfo.RSS
				taskMetric.VirtualMemoryBytes = memInfo.VMS
			}

			if ioStats != nil {
				taskMetric.ReadBytes = ioStats.ReadBytes
				taskMetric.WriteBytes = ioStats.WriteBytes
			}

			processMetrics.TaskProcesses = append(processMetrics.TaskProcesses, taskMetric)
		}
	}
	mc.taskMutex.RUnlock()

	return processMetrics
}

// collectLoadMetrics collects system load metrics
func (mc *MetricsCollector) collectLoadMetrics() *pb.LoadMetrics {
	loadAvg, err := load.Avg()
	if err != nil {
		log.Printf("Failed to get load average: %v", err)
		return &pb.LoadMetrics{}
	}

	return &pb.LoadMetrics{
		Load1:  loadAvg.Load1,
		Load5:  loadAvg.Load5,
		Load15: loadAvg.Load15,
	}
}

// GetSystemInfo collects and returns system information
func (mc *MetricsCollector) GetSystemInfo() *pb.SystemInfoResponse {
	return &pb.SystemInfoResponse{
		Hardware:          mc.collectHardwareInfo(),
		Software:          mc.collectSoftwareInfo(),
		NetworkInterfaces: mc.collectNetworkInfo(),
		Timestamp:         uint64(time.Now().Unix()),
	}
}

// collectHardwareInfo collects hardware information
func (mc *MetricsCollector) collectHardwareInfo() *pb.SystemHardware {
	hardware := &pb.SystemHardware{
		Architecture: runtime.GOARCH,
		CoreCount:    uint32(runtime.NumCPU()),
		ThreadCount:  uint32(runtime.NumCPU()), // Go doesn't distinguish between cores and threads easily
	}

	// CPU information
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info := cpuInfo[0]
		hardware.Cpu = &pb.CpuInfo{
			Model:            info.ModelName,
			Cores:            uint32(info.Cores),
			Threads:          uint32(runtime.NumCPU()),
			BaseFrequencyGhz: info.Mhz / 1000.0,
			MaxFrequencyGhz:  info.Mhz / 1000.0, // gopsutil doesn't provide max frequency
			Features:         info.Flags,
		}
	}

	// Memory information
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		hardware.Memory = &pb.MemoryInfo{
			TotalBytes:     memInfo.Total,
			AvailableBytes: memInfo.Available,
			// Memory type and speed are not easily available via gopsutil
			MemoryType: "Unknown",
			SpeedMhz:   0,
		}
	}

	// Disk information
	partitions, err := disk.Partitions(false)
	if err == nil {
		for _, partition := range partitions {
			usage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				continue
			}

			diskInfo := &pb.DiskInfo{
				Device:         partition.Device,
				MountPoint:     partition.Mountpoint,
				Filesystem:     partition.Fstype,
				TotalBytes:     usage.Total,
				AvailableBytes: usage.Free,
				DiskType:       "Unknown", // gopsutil doesn't provide disk type easily
			}

			hardware.Disks = append(hardware.Disks, diskInfo)
		}
	}

	return hardware
}

// collectSoftwareInfo collects software information
func (mc *MetricsCollector) collectSoftwareInfo() *pb.SystemSoftware {
	hostInfo, err := host.Info()
	if err != nil {
		log.Printf("Failed to get host info: %v", err)
		return &pb.SystemSoftware{}
	}

	software := &pb.SystemSoftware{
		Hostname:      hostInfo.Hostname,
		UptimeSeconds: hostInfo.Uptime,
		Os: &pb.OperatingSystem{
			Name:          hostInfo.OS,
			Version:       hostInfo.PlatformVersion,
			KernelVersion: hostInfo.KernelVersion,
			Distribution:  hostInfo.Platform,
		},
		// Package information would require OS-specific implementations
		Packages: []*pb.InstalledPackage{},
	}

	return software
}

// collectNetworkInfo collects network interface information
func (mc *MetricsCollector) collectNetworkInfo() []*pb.NetworkInterface {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Failed to get network interfaces: %v", err)
		return []*pb.NetworkInterface{}
	}

	var networkInterfaces []*pb.NetworkInterface
	for _, iface := range interfaces {
		netInterface := &pb.NetworkInterface{
			Name:       iface.Name,
			MacAddress: iface.HardwareAddr,
			IsUp:       strings.Contains(iface.Flags[0], "up"),
			// Speed information is not easily available
			SpeedMbps: 0,
		}

		// Add IP addresses
		for _, addr := range iface.Addrs {
			netInterface.IpAddresses = append(netInterface.IpAddresses, addr.Addr)
		}

		networkInterfaces = append(networkInterfaces, netInterface)
	}

	return networkInterfaces
}
