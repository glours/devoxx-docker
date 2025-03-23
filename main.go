package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/rumpl/devoxx-docker/cgroups"
	"github.com/rumpl/devoxx-docker/remote"
)

func main() {
	switch os.Args[1] {
	case "pull":
		if err := pull(os.Args[2]); err != nil {
			log.Fatal(err)
		}
	case "run":
		if err := run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "child":
		if err := child(os.Args[2], os.Args[3], os.Args[4:]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Unknown command %s\n", os.Args[1])
	}
}

func pull(image string) error {
	fmt.Printf("Pulling %s\n", image)
	puller := remote.NewImagePuller(image)
	err := puller.Pull()
	fmt.Println("Pulled image")
	return err
}

func run(args []string) error {
	imageName := args[0]
	_, err := os.Stat("/fs/" + imageName)
	if err != nil {
		if os.IsNotExist(err) {
			if err := pull(imageName); err != nil {
				return fmt.Errorf("pull %w", err)
			}
		} else {
			return err
		}
	}

	if err := os.MkdirAll("/fs/"+imageName+"/etc", 0755); err != nil {
		return fmt.Errorf("create etc dir: %w", err)
	}

	if err := os.WriteFile("/fs/"+imageName+"/etc/resolv.conf", []byte("nameserver 1.1.1.1\n"), 0644); err != nil {
		return fmt.Errorf("write resolv.conf: %w", err)
	}

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	// Create veth pair
	vethName := "veth0"
	peerName := "veth1"
	if err := exec.Command("ip", "link", "add", vethName, "type", "veth", "peer", "name", peerName).Run(); err != nil {
		return fmt.Errorf("create veth pair %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %w", err)
	}

	// Move peer end into container network namespace
	if err := exec.Command("ip", "link", "set", peerName, "netns", fmt.Sprintf("%d", cmd.Process.Pid)).Run(); err != nil {
		return fmt.Errorf("move veth to netns %w", err)
	}

	// Setup host end
	if err := exec.Command("ip", "addr", "add", "10.0.0.1/24", "dev", vethName).Run(); err != nil {
		return fmt.Errorf("add ip to host veth %w", err)
	}
	if err := exec.Command("ip", "link", "set", vethName, "up").Run(); err != nil {
		return fmt.Errorf("set host veth up %w", err)
	}

	// Setup NAT
	if err := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "10.0.0.0/24", "-j", "MASQUERADE").Run(); err != nil {
		return fmt.Errorf("setup NAT %w", err)
	}

	if err := cgroups.SetupCgroup(cmd.Process.Pid); err != nil {
		return fmt.Errorf("setup cgroup %w", err)
	}

	err = cmd.Wait()

	// Cleanup NAT rule
	if err := exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-s", "10.0.0.0/24", "-j", "MASQUERADE").Run(); err != nil {
		log.Printf("Warning: failed to cleanup NAT rule: %v", err)
	}

	// Cleanup veth pair
	if err := exec.Command("ip", "link", "delete", vethName).Run(); err != nil {
		log.Printf("Warning: failed to cleanup veth pair: %v", err)
	}

	if err := cgroups.RemoveCgroup(cmd.Process.Pid); err != nil {
		log.Printf("Warning: failed to remove cgroup: %v", err)
	}

	return err
}

func child(image string, command string, args []string) error {
	fmt.Printf("Running %s in %s\n", command, image)

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	volumeDestination := fmt.Sprintf("/fs/%s/volume", image)
	if err := os.MkdirAll(volumeDestination, 0755); err != nil {
		return fmt.Errorf("mkdir %w", err)
	}

	if err := syscall.Mount("/workspaces/devoxx-docker/volume", volumeDestination, "", syscall.MS_PRIVATE|syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("mount volume %w", err)
	}

	if err := syscall.Chroot("/fs/" + image); err != nil {
		return fmt.Errorf("chroot %w", err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir %w", err)
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount proc %w", err)
	}

	if err := syscall.Mount("sysfs", "/sys", "sysfs", 0, ""); err != nil {
		return fmt.Errorf("mount sys %w", err)
	}
	if err := syscall.Mount("cgroup2", "/sys/fs/cgroup", "cgroup2", 0, ""); err != nil {
		return fmt.Errorf("mount cgroup2 %w", err)
	}

	if err := syscall.Mount("dev", "/dev", "devtmpfs", 0, ""); err != nil {
		return fmt.Errorf("mount dev %w", err)
	}

	if err := syscall.Sethostname([]byte("devoxx-container")); err != nil {
		return fmt.Errorf("set hostname %w", err)
	}

	peerName := "veth1"
	if err := exec.Command("ip", "addr", "add", "10.0.0.2/24", "dev", peerName).Run(); err != nil {
		return fmt.Errorf("add ip to peer %w", err)
	}
	if err := exec.Command("ip", "link", "set", peerName, "up").Run(); err != nil {
		return fmt.Errorf("set peer up %w", err)
	}
	if err := exec.Command("ip", "route", "add", "default", "via", "10.0.0.1").Run(); err != nil {
		return fmt.Errorf("add route %w", err)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start %w", err)
	}

	if err := syscall.Unmount("/proc", 0); err != nil {
		return fmt.Errorf("unmount proc %w", err)
	}

	if err := syscall.Unmount("/sys/fs/cgroup", 0); err != nil {
		return fmt.Errorf("unmount cgroup %w", err)
	}

	if err := syscall.Unmount("/sys", 0); err != nil {
		return fmt.Errorf("unmount sys %w", err)
	}

	if err := syscall.Unmount("/dev", 0); err != nil {
		return fmt.Errorf("unmount dev %w", err)
	}

	return nil
}
