# Implementing Namespace Isolation

## Objective

In this exercise, you will learn how to implement namespace isolation and hostname configuration for containers. You will focus on creating a new PID namespace and setting a custom hostname using the UTS namespace.


## Steps

### Step 1: Add Namespace Isolation

1. Modify the parent process creation to include namespace flags:
    ```go
    func run() error {
        cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args...)...)
        
         //TODO: 
        // 1. Add namespace flags for PID and UTS namespaces
		
        if err := cmd.Wait(); err != nil {
           return fmt.Errorf("wait %w", err)
	    }

	    fmt.Printf("Container exited with exit code %d\n", cmd.ProcessState.ExitCode())
    }
    ```
<details>
<summary>Hint</summary>
look at `syscall.SysProcAttr` structure
</details>

### Step 2: Implement Hostname Changes

1. Add hostname configuration to the child process:
    ```go
    func child() error {
        //TODO: 
		// 1. Set the container hostname
		// 2.Print the child PID 
		// 3.Print the hostname to verify the change
    }
    ```
<details>
<summary>Hint</summary>
look at `syscall.Sethostname` function
</details>

### Step 4: Testing

1. Build and run your program:
    ```bash
    # Build the program
    make

    # Run with sudo (needed for namespace operations)
    sudo ./bin/devoxx-container
   ```

### Summary

We have now implemented PID and UTS namespace isolation, providing process isolation and custom hostname configuration for containers.   
This is a crucial step towards building a fully functional container runtime.


[Next step](04-namespaces-and-chroot.md)

## Hints

- Use `syscall.CLONE_NEWPID` for PID namespace isolation
- Use `syscall.CLONE_NEWUTS` for hostname namespace isolation
- Root privileges are required for namespace operations
- The child process should have PID 1 in its namespace

## Key Points

- PID namespace provides process isolation
- UTS namespace enables custom hostname
- Namespace changes require root privileges
- Child process sees itself as PID 1

## Additional Resources

- [man namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [man clone](https://man7.org/linux/man-pages/man2/clone.2.html)
- [Go syscall package](https://pkg.go.dev/syscall)

## Command Reference

### Namespace Operations
```go
// Create new namespaces
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
}

// Set hostname
syscall.Sethostname([]byte("new-hostname"))
```

### Debugging Commands
```bash
# Check process namespaces
ls -l /proc/$$/ns/

# View hostname
hostname

# Check PID in different namespaces
ps aux
```

### Error Handling Examples
```go
// Handle hostname errors
if err := syscall.Sethostname([]byte("container-host")); err != nil {
    if os.IsPermission(err) {
        return fmt.Errorf("permission denied: run with sudo: %w", err)
    }
    return fmt.Errorf("hostname error: %w", err)
}
```