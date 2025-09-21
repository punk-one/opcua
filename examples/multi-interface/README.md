# OPC UA Connection Example in Multi-Interface Environment

This example demonstrates how to connect to different OPC UA devices by specifying network interfaces in a multi-interface environment.

## Use Cases

When your computer has multiple network interfaces and you need to connect to different OPC UA devices through different network cards, this functionality is very useful.

**Typical Scenario:**
- Computer has two network cards: `192.168.100.10` and `192.168.100.20`
- Both network cards are connected to OPC UA devices with IP `192.168.100.1`
- Need to connect to the corresponding devices through specified network cards

## Main Features

### New Configuration Options

```go
// LocalAddr sets the local address to bind when connecting
// This allows specifying which network interface to use for connection
// Example: "192.168.100.10:0" uses the network interface with IP 192.168.100.10
opcua.LocalAddr("192.168.100.10:0")
```

### Usage

```go
// Connect through the first network card
client1, err := opcua.NewClient("opc.tcp://192.168.100.1:4840", 
    opcua.LocalAddr("192.168.100.10:0"),  // Specify local network card
    opcua.SecurityPolicy(ua.SecurityPolicyURINone),
    opcua.SecurityModeString("None"),
)

// Connect through the second network card
client2, err := opcua.NewClient("opc.tcp://192.168.100.1:4840", 
    opcua.LocalAddr("192.168.100.20:0"),  // Specify another local network card
    opcua.SecurityPolicy(ua.SecurityPolicyURINone),
    opcua.SecurityModeString("None"),
)
```

## Running the Example

```bash
# Basic usage
go run multi-interface.go

# Specify custom parameters
go run multi-interface.go \
    -endpoint1="opc.tcp://192.168.100.1:4840" \
    -endpoint2="opc.tcp://192.168.100.1:4840" \
    -local1="192.168.100.10:0" \
    -local2="192.168.100.20:0" \
    -node="i=2258"
```

## Parameter Description

- `-endpoint1`: First OPC UA server endpoint
- `-endpoint2`: Second OPC UA server endpoint  
- `-local1`: Local network card address to use when connecting to the first device
- `-local2`: Local network card address to use when connecting to the second device
- `-node`: Node ID to read

## Network Configuration Requirements

1. **Ensure Network Card Configuration is Correct**
   ```bash
   # View network interfaces on Linux/Windows
   ip addr show    # Linux
   ipconfig        # Windows
   ```

2. **Ensure Routing Configuration is Correct**
   ```bash
   # Add routing rules (if needed)
   route add -net 192.168.100.0/24 gw 192.168.100.1 dev eth0
   ```

3. **Firewall Configuration**
   Ensure the firewall allows OPC UA communication (default port 4840)

## Troubleshooting

### Common Errors

1. **"bind: cannot assign requested address"**
   - Check if the specified local address exists in the system
   - Ensure the network card is properly configured and enabled

2. **"no route to host"**
   - Check network routing configuration
   - Ensure the target device is reachable

3. **Connection timeout**
   - Check firewall settings
   - Verify that the OPC UA server is running
   - Confirm the port number is correct

### Debugging Tips

1. **Use ping to test network connectivity**
   ```bash
   # Ping target device from specified network card
   ping -I 192.168.100.10 192.168.100.1
   ping -I 192.168.100.20 192.168.100.1
   ```

2. **Use telnet to test port connectivity**
   ```bash
   telnet 192.168.100.1 4840
   ```

3. **Enable debug logging**
   ```go
   // Add debug output in code
   debug.Enable = true
   ```

## Advanced Usage

### Custom Dialer

If more fine-grained network control is needed, you can create a custom Dialer:

```go
import (
    "net"
    "github.com/gopcua/opcua/uacp"
)

// Create custom Dialer
customDialer := &uacp.Dialer{
    Dialer: &net.Dialer{
        LocalAddr: &net.TCPAddr{
            IP: net.ParseIP("192.168.100.10"),
        },
        Timeout: 30 * time.Second,
    },
}

// Use custom Dialer
client, err := opcua.NewClient(endpoint, 
    opcua.Dialer(customDialer),
    // Other options...
)
```

### Concurrent Connections to Multiple Devices

The example code includes examples of how to monitor multiple devices simultaneously, see the `monitorMultipleDevices` function.

## Notes

1. **Port Binding**: Use `:0` as the port number to let the system automatically assign an available port
2. **Resource Management**: Ensure client connections are properly closed to release network resources
3. **Error Handling**: Network connections may be unstable, implement appropriate error handling and reconnection mechanisms
4. **Performance Considerations**: Multiple concurrent connections will consume more system resources, adjust according to actual needs