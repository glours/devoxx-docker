# Adding Network Support to Containers

## Objective

In this exercise, you will implement network isolation and connectivity for containers using virtual ethernet (veth) pairs and network namespaces. This will enable containers to communicate with the host and access the internet while maintaining network isolation.

## Steps

### Step 1: Create Network Namespace

1. Add network namespace isolation in your main container creation code:
    ```go
    cmd.SysProcAttr = &syscall.SysProcAttr{
        // Add CLONE_NEWNET to your existing clone flags
        // This creates a new network namespace for the container
    }
    ```

### Step 2: Implement veth Pair Creation

1. Create a function to set up the veth pair:
    ```go
    func SetupVeth(vethName string, pid int) error {
        // TODO: Use "ip link" commands to:
        // 1. Create a veth pair (veth0 and veth1)
        // 2. Move veth1 to the container's network namespace
        // 3. Configure veth0 in the host namespace
        // 4. Set up NAT rules using iptables
    }
    ```

2. Create a cleanup function to remove the network configuration:
    ```go
    func CleanupVeth(vethName string) error {
        // TODO: Clean up:
        // 1. Remove NAT rules
        // 2. Delete the veth pair
    }
    ```

### Step 3: Configure Container Networking

1. Create a function to set up networking inside the container:
    ```go
    func SetupContainerNetworking(peerName string) error {
        // TODO: Inside the container:
        // 1. Assign IP address to the container interface
        // 2. Bring up the interface
        // 3. Set up the default route
        // 4. Configure the loopback interface
    }
    ```

### Step 4: Integration

1. Add network setup to your container creation flow:
    ```go
    // After starting the container process:
    vethName := "veth0"
    if err := SetupVeth(vethName, cmd.Process.Pid); err != nil {
        return err
    }
    defer CleanupVeth(vethName)  // Ensure cleanup on exit
    ```

### Step 5: Testing

1. Test your network implementation:
    ```bash
    # From inside the container
    ping 10.0.0.1     # Should reach host
    ping 8.8.8.8      # Should reach internet
    ping google.com   # Should resolve and reach
    ```

## Hints

- Use `exec.Command()` to execute network configuration commands
- The container interface should be in the 10.0.0.0/24 subnet
- Common IP assignments:
  - Host interface (veth0): 10.0.0.1
  - Container interface (veth1): 10.0.0.2
- Required iptables rules should enable NAT for the container subnet
- Don't forget to enable IP forwarding on the host
- Use `defer` for cleanup operations

## Key Points

- Network namespaces provide network isolation
- veth pairs create a virtual network connection
- NAT enables internet access from the container
- Proper cleanup is essential to avoid resource leaks

## Additional Resources

- [man ip-netns](https://man7.org/linux/man-pages/man8/ip-netns.8.html)
- [man veth](https://man7.org/linux/man-pages/man4/veth.4.html)
- [man iptables](https://man7.org/linux/man-pages/man8/iptables.8.html)
- [Linux Network Namespaces](https://man7.org/linux/man-pages/man7/network_namespaces.7.html)
- [Container Networking](https://docs.docker.com/network/)

## Required Commands Reference

### Host Network Configuration
```bash
# Create veth pair
ip link add <veth-host> type veth peer name <veth-container>

# Move container end to container namespace
ip link set <veth-container> netns <pid>

# Configure host end
ip addr add 10.0.0.1/24 dev <veth-host>
ip link set <veth-host> up

# Configure NAT
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -j MASQUERADE
```

### Container Network Configuration
```bash
# Configure container interface
ip addr add 10.0.0.2/24 dev <veth-container>
ip link set <veth-container> up
ip link set lo up
ip route add default via 10.0.0.1
```