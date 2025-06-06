package plugininterceptor

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"plugin"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/metadata"
)

// TODO(nikolabo): synchronize access to these?
var currentClientChain grpc.UnaryClientInterceptor
var currentServerChain grpc.UnaryServerInterceptor
var highestInterceptorFile string

// var highestLBFile string
var InterceptorPluginPrefix string
var LBPluginPrefix string
var pluginInterface interceptInit
var versionNumber int
var versionNumberLock sync.RWMutex
var sharedData = &sync.Map{}

type interceptInit interface {
	ClientInterceptor() grpc.UnaryClientInterceptor
	ServerInterceptor() grpc.UnaryServerInterceptor
	Kill() // call to disable weak synchronization goroutine in plugin
}

func init() {
	go func() {
		filePath := "/appnet/config-version"
		for {
			updateVersionNumberFromFile(filePath)
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	go func() {
		for {
			if InterceptorPluginPrefix != "" || LBPluginPrefix != "" {
				updateChains(InterceptorPluginPrefix)
				// updateLB(LBPluginPrefix)
			}
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	// Register the default appnet balancer (round robin)
	balancer.Register(NewBuilder(sharedData))
}

func ClientInterceptor(InterceptorPluginPrefixPath, LBPluginPrefixPath string) grpc.UnaryClientInterceptor {
	fmt.Println("ClientInterceptor called with", InterceptorPluginPrefixPath, LBPluginPrefixPath)
	// Interceptor and lb plugins should be compiled/updated at the same time
	if InterceptorPluginPrefix != InterceptorPluginPrefixPath || LBPluginPrefix != LBPluginPrefixPath {
		updateChains(InterceptorPluginPrefixPath)
		// updateLB(LBPluginPrefixPath)
	}
	InterceptorPluginPrefix = InterceptorPluginPrefixPath
	LBPluginPrefix = LBPluginPrefixPath
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Add unique id to rpcs
		rpc_id := rand.Uint32()
		ctx = metadata.AppendToOutgoingContext(ctx, "appnet-rpc-id", strconv.FormatUint(uint64(rpc_id), 10))

		// temp hack
		ctx = metadata.AppendToOutgoingContext(ctx, "shard-key", strconv.FormatUint(rand.Uint64N(800), 10))

		// Add config-version header
		ctx = metadata.AppendToOutgoingContext(ctx, "appnet-config-version", strconv.Itoa(getVersionNumber()))

		// ------------------------------------------------------------------------------------------------

		// Extract metadata from the outgoing context
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		// Check for the "key" header and process it
		if keys, exists := md["key"]; exists && len(keys) > 0 {
			key := keys[0]
			// Match the pattern "header-xxx"
			re := regexp.MustCompile(`^header-(\d+)$`)
			matches := re.FindStringSubmatch(key)
			if len(matches) == 2 {
				// Parse the number of headers to add
				numHeaders, err := strconv.Atoi(matches[1])
				if err == nil && numHeaders > 0 {
					// Generate random headers
					for i := 0; i < numHeaders; i++ {
						headerName := "appnet-header-" + strconv.FormatUint(rand.Uint64N(10000), 10)
						headerValue := "appnet-value-" + strconv.FormatUint(rand.Uint64N(10000), 10)
						ctx = metadata.AppendToOutgoingContext(ctx, headerName, headerValue)
					}
				}
			}
		}

		// ------------------------------------------------------------------------------------------------

		if currentClientChain == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		return currentClientChain(ctx, method, req, reply, cc, invoker, opts...)
	}
}

func ServerInterceptor(InterceptorPluginPrefixPath string) grpc.UnaryServerInterceptor {
	fmt.Println("ServerInterceptor called with", InterceptorPluginPrefixPath)
	if InterceptorPluginPrefix != InterceptorPluginPrefixPath {
		updateChains(InterceptorPluginPrefixPath)
	}
	InterceptorPluginPrefix = InterceptorPluginPrefixPath
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)

		log.Println("Incoming Metadata (Headers):")
		for key, values := range md {
			log.Printf("  %s: %v", key, values)
		}

		if rpcID, exists := md["appnet-rpc-id"]; exists && len(rpcID) > 0 {
			headers := metadata.Pairs(
				"appnet-rpc-id", rpcID[0], // Include "appnet-rpc-id" only if it exists
				"grpc-status", "0", // This works as normal
			)

			// ------------------------------------------------------------------------------------------------

			// Include all headers that match the "appnet-header-xxx" pattern from the request
			for key, values := range md {
				if len(values) > 0 && strings.HasPrefix(key, "appnet-header") {
					headers.Append(key, values[0])
				}
			}

			// Print headers before sending
			log.Println("Headers to be sent:")
			for key, values := range headers {
				log.Printf("  %s: %v", key, values)
			}

			// ------------------------------------------------------------------------------------------------

			grpc.SendHeader(ctx, headers)
		}

		if currentServerChain == nil {
			return handler(ctx, req)
		}

		return currentServerChain(ctx, req, info, handler)
	}
}

func getVersionNumber() int {
	versionNumberLock.RLock()
	defer versionNumberLock.RUnlock()
	return versionNumber
}

func updateVersionNumberFromFile(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			versionNumberLock.Lock()
			defer versionNumberLock.Unlock()
			versionNumber = -1
		} else {
			fmt.Println("Error reading file:", err)
		}
		return
	}

	trimmedData := strings.TrimSpace(string(data))
	newVersion, err := strconv.Atoi(trimmedData)
	if err != nil {
		fmt.Println("Error converting file content to int:", err)
		return
	}

	versionNumberLock.Lock()
	defer versionNumberLock.Unlock()

	if versionNumber != newVersion {
		versionNumber = newVersion
	}
}

// func updateLB(prefix string) {
// 	var highestSeenLB string = highestLBFile

// 	dir, prefix := filepath.Split(prefix)
// 	files, _ := os.ReadDir(dir)

// 	for _, file := range files {
// 		if strings.HasPrefix(file.Name(), prefix) {
// 			if file.Name() > highestSeenLB {
// 				highestSeenLB = file.Name()
// 			}
// 		}
// 	}

// 	if highestSeenLB != highestLBFile {
// 		highestLBFile = highestSeenLB
// 		loadLoadBalancerPlugin(dir + highestLBFile)
// 	}
// }

func updateChains(prefix string) {
	// fmt.Printf("updateChains called with %s\n", prefix)
	var highestSeenInterceptor string = highestInterceptorFile

	dir, prefix := filepath.Split(prefix)
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) {
			if file.Name() > highestSeenInterceptor {
				highestSeenInterceptor = file.Name()
			}
		}
	}

	if highestSeenInterceptor != highestInterceptorFile {
		highestInterceptorFile = highestSeenInterceptor
		intercept := loadInterceptorsPlugin(dir + highestInterceptorFile)
		if pluginInterface != nil {
			pluginInterface.Kill()
		}
		pluginInterface = intercept
		currentClientChain = intercept.ClientInterceptor()
		currentServerChain = intercept.ServerInterceptor()
	}
}

func loadInterceptorsPlugin(interceptorPluginPath string) interceptInit {
	// TODO: return err instead of panicking
	fmt.Printf("Loading plugin: %s\n", interceptorPluginPath)
	interceptorPlugin, err := plugin.Open(interceptorPluginPath)
	if err != nil {
		fmt.Printf("loading error: %v\n", err)
		panic("error loading interceptor plugin so")
	}

	symInterceptInit, err := interceptorPlugin.Lookup("InterceptInit")
	if err != nil {
		panic("error locating interceptor in plugin so")
	}

	interceptInit, ok := symInterceptInit.(interceptInit)
	if !ok {
		panic("error casting interceptInit")
	}

	fmt.Printf("Loaded plugin: %s\n", interceptorPluginPath)
	return interceptInit
}

// func loadLoadBalancerPlugin(lbPluginPath string) {
// 	// TODO: return err instead of panicking
// 	p, err := plugin.Open(lbPluginPath)
// 	if err != nil {
// 		fmt.Printf("loading error: %v\n", err)
// 		panic("error loading load balancer plugin so")
// 	}

// 	// Lookup the NewBuilder symbol (function)
// 	symbol, err := p.Lookup("NewBuilder")
// 	if err != nil {
// 		panic("error locating NewBuilder in plugin so")
// 	}

// 	// Assert that the symbol is of the correct type (function with expected signature)
// 	newBuilderFunc, ok := symbol.(func(*sync.Map) balancer.Builder)
// 	if !ok {
// 		panic("error casting NewBuilder")
// 	}

// 	fmt.Printf("Loaded balancer plugin: %s\n", lbPluginPath)
// 	balancer.Register(newBuilderFunc(sharedData))
// }
