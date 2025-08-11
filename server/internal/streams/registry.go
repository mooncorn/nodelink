package streams

// // StreamType defines the configuration for different types of streams
// type StreamType struct {
// 	Name         string `json:"name"`
// 	Description  string `json:"description"`
// 	Buffered     bool   `json:"buffered"`
// 	BufferSize   int    `json:"buffer_size"`
// 	AutoCleanup  bool   `json:"auto_cleanup"`
// 	ProcessorKey string `json:"processor_key"`
// }

// // StreamTypes registry of all available stream types
// var StreamTypes = map[string]StreamType{
// 	"shell_output": {
// 		Name:         "shell_output",
// 		Description:  "Shell command output streaming",
// 		Buffered:     true,
// 		BufferSize:   50,
// 		AutoCleanup:  true,
// 		ProcessorKey: "shell",
// 	},
// 	"metrics": {
// 		Name:         "metrics",
// 		Description:  "System metrics streaming",
// 		Buffered:     true,
// 		BufferSize:   100,
// 		AutoCleanup:  false,
// 		ProcessorKey: "metrics",
// 	},
// 	"container_logs": {
// 		Name:         "container_logs",
// 		Description:  "Container log streaming",
// 		Buffered:     true,
// 		BufferSize:   200,
// 		AutoCleanup:  true,
// 		ProcessorKey: "docker",
// 	},
// 	"docker_operations": {
// 		Name:         "docker_operations",
// 		Description:  "Docker operation status",
// 		Buffered:     false,
// 		BufferSize:   0,
// 		AutoCleanup:  true,
// 		ProcessorKey: "docker",
// 	},
// }

// // GetStreamType returns a stream type by name
// func GetStreamType(name string) (StreamType, bool) {
// 	streamType, exists := StreamTypes[name]
// 	return streamType, exists
// }

// // GetAllStreamTypes returns all available stream types
// func GetAllStreamTypes() map[string]StreamType {
// 	return StreamTypes
// }

// // RegisterStreamType adds a new stream type to the registry
// func RegisterStreamType(name string, streamType StreamType) {
// 	StreamTypes[name] = streamType
// }
