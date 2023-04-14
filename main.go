package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func GetHostsFilePath() (string, error) {
	hosts_dst_file := "/etc/hosts"
	if runtime.GOOS == "windows" {
		win_sys := os.Getenv("windir")
		hosts_dst_file := filepath.Join(win_sys, "System32", "drivers", "etc", "hosts")
		_, err := os.Stat(hosts_dst_file)
		if os.IsExist(err) == false {
			return hosts_dst_file, err
		}
		return "", err
	} else if runtime.GOOS == "linux" {
		_, err := os.Stat(hosts_dst_file)
		if os.IsExist(err) == false {
			return hosts_dst_file, err
		}
		return "", err
	} else if runtime.GOOS == "darwin" {
		_, err := os.Stat(hosts_dst_file)
		if os.IsExist(err) == false {
			return hosts_dst_file, err
		}
		return "", err
	} else {
		return "", nil
	}
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func BackupHostsFile(hosts_file string) (bool, error) {
	backup_ok := true
	hosts_file_bak := hosts_file + ".bak"
	// _, err := os.Stat(hosts_file_bak)
	if _, err := os.Stat(hosts_file_bak); err != nil {
		if os.IsExist(err) {
			return backup_ok, err
		}
	}
	_, err := CopyFile(hosts_file_bak, hosts_file)
	if err == nil {
		return backup_ok, err
	}
	return false, err
}

func IsAdminRunning() bool {
	if runtime.GOOS == "windows" {
		_, err := exec.Command("net", "session").Output()
		if err != nil {
			log.Panic("Please start the WebHostAcceleratorForHosts as administrator!")
			os.Exit(1)
		}
		return true
	} else {
		if os.Getuid() != 0 {
			log.Panic("Please start the WebHostAcceleratorForHosts as root or sudo!")
			os.Exit(1)
		}
		return true
	}

}

func HijackGithubHosts(hosts_file string, github_host string, domain string) {
	content, err := ioutil.ReadFile(hosts_file)
	if err != nil {
		log.Panic("WebHostAcceleratorForHosts Read Hosts ERROR:", err)
		return
	}
	var lines []string
	if runtime.GOOS == "windows" {
		lines = strings.Split(string(content), "\r\n")
	} else {
		lines = strings.Split(string(content), "\n")
	}

	is_exist := false
	var new_lines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "#") {
			new_lines = append(new_lines, line)
			continue
		}
		parts := strings.Fields(line)
		d := parts[1]
		if d == domain {
			is_exist = true
			parts[0] = github_host
			new_lines = append(new_lines, strings.Join(parts, " "))
			continue
		}
		new_lines = append(new_lines, line)
	}
	if !is_exist {
		// _, ip_net, err := net.ParseCIDR(github_host)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }
		new_lines = append(new_lines, github_host+"	"+domain)
	}

	file, err := os.OpenFile(hosts_file, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, line := range new_lines {
		if runtime.GOOS == "windows" {
			_, err = writer.WriteString(line + "\r\n")
			if err != nil {
				fmt.Println(err)
				return
			}
		} else {
			_, err = writer.WriteString(line + "\n")
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
	writer.Flush()
}

func OpenLocalWebBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func ModHostsForGithubCom(ipaddr string) {
	hosts_file, err := GetHostsFilePath()
	if err != nil {
		log.Panic("WebHostAcceleratorForHosts Find Hosts ERROR:", err)
		return
	}
	_, err = BackupHostsFile(hosts_file)
	if err != nil {
		log.Panic("WebHostAcceleratorForHosts Bakcup Hosts ERROR:", err)
		return
	}
	log.Print("[+] WebHostAcceleratorForHosts Backup Hosts File Success:", hosts_file)
	HijackGithubHosts(hosts_file, ipaddr, "github.com")
}

type dns struct {
	ip            string
	indexPriority int
}

type DNSList []dns

// PrintDNSList 会打印出IP地址和优先级索引
func (list DNSList) PrintDNSList() {
	log.Print("[+] IP address\tIndex priority")
	for _, ip := range list {
		log.Printf("\t%s\t%d\n", ip.ip, ip.indexPriority)
	}
}

// GetSelectedIPByPriority 会查找用户选择的IP地址
func (list DNSList) GetSelectedIPByPriority(selected string) (string, error) {
	var selectedIP dns
	for _, ip := range list {
		if selected == fmt.Sprintf("%d", ip.indexPriority) {
			selectedIP = ip
			break
		}
	}
	if selectedIP.ip == "" {
		return "", fmt.Errorf("invalid index selected")
	}
	return selectedIP.ip, nil
}

func ExecCommand(command string, args ...string) (string, error) {
	// 创建一个Cmd结构体
	cmd := exec.Command(command, args...)

	// 输出命令行日志
	fmt.Println("Execute command: ", cmd)

	// 创建一个bytes.Buffer类型的指针，捕获标准输出
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// 创建一个bytes.Buffer类型的指针，捕获标准错误输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// 运行命令
	err := cmd.Run()

	// 如果命令运行时发生错误，返回错误消息和标准错误输出
	if err != nil {
		return stderr.String(), err
	}

	// 返回标准输出，没有错误
	return stdout.String(), nil
}

func main() {
	IsAdminRunning()
	if len(os.Args) != 2 {
		ips := DNSList{
			{ip: "140.82.121.4", indexPriority: 1},
			{ip: "20.205.243.166", indexPriority: 2},
			{ip: "20.248.137.48", indexPriority: 3},
			{ip: "140.82.121.4", indexPriority: 4},
			{ip: "20.27.177.113", indexPriority: 5},
		}

		// 显示所有IP地址及其优先级索引
		ips.PrintDNSList()

		// 请求用户输入选择的索引
		var selected string
		fmt.Print("[-] Please enter an index: ")
		_, err := fmt.Scanln(&selected)
		if err != nil {
			log.Panic("Error reading input:", err)
			return
		}

		// 返回用户选择的IP地址
		selectedIP, err := ips.GetSelectedIPByPriority(selected)
		if err != nil {
			log.Panic(err)
			return
		}
		log.Print("[+] Selected IP address:", selectedIP)
		ModHostsForGithubCom(selectedIP)
		return
	} else {
		OpenLocalWebBrowser("https://tool.chinaz.com/speedworld/github.com")
		log.Printf("[+] Open https://tool.chinaz.com/speedworld/github.com")
		ModHostsForGithubCom(os.Args[1])
	}

	if runtime.GOOS != "windows" {
		cmd := exec.Command("sudo", "killall", "-9", "mDNSResponder", "mDNSResponderHelper")

		// 输出命令字符串并运行
		log.Print("[+] Execute command:", strings.Join(cmd.Args, " "))
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok { // 处理exit code非0的情况
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					log.Fatalf("Command failed with exit code %d\n", status.ExitStatus())
					return
				}
			}
			log.Print("[+] Command execution failed:", err)
		}
		log.Print("[+] Command executed successfully")
	}
}
