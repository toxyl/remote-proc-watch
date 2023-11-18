package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/toxyl/glog"
)

const (
	WP        = 20
	WHOST     = 32
	WPID      = 10
	WCMD      = 35
	WCPU      = 55
	WCPU_HEAD = WCPU - 20
	WMEM      = 65
	WMEM_HEAD = WMEM - 19
	WMEMB     = 15
)

var log = glog.NewLoggerSimple("rpw")

func printHeader(host, pid, cmd, cpu, mem string, wHost, wPid, wCmd, wCpu, wMem, wMemB int) {
	log.Blank(
		glog.Bold()+"\033[97;40m%s %s %s %s %s %s"+glog.Reset(),
		glog.PadRight(host, wHost, ' '),
		glog.PadRight(pid, wPid, ' '),
		glog.PadRight(cmd, wCmd, ' '),
		glog.PadRight(cpu, wCpu, ' '),
		glog.PadRight(mem, wMem-wMemB, ' '),
		glog.PadRight("", wMemB, ' '),
	)
}

func printRow(host, pid, cmd string, cpu, mem, memB float64, wHost, wPid, wCmd, wCpu, wMem, wMemB, wP int) {
	log.Blank(
		"%s %s %s %s %s %s",
		glog.PadRight(glog.Auto(host), wHost, ' '),
		glog.PadRight(glog.Auto(pid), wPid, ' '),
		glog.PadRight(glog.Auto(cmd), wCmd, ' '),
		glog.PadRight(glog.ProgressBar(cpu/100.0, wP), wCpu, ' '),
		glog.PadRight(glog.ProgressBar(mem/100.0, wP), wMem-wMemB, ' '),
		glog.PadLeft(glog.HumanReadableBytesIEC(memB*1024), wMemB, ' '),
	)
}

func printFooter(cpu, mem, memB float64, wHost, wPid, wCmd, wCpu, wMem, wMemB, wP, numHosts int) {
	log.Blank(
		glog.Bold()+"\033[97;40m%s %s %s"+glog.Reset()+" %s %s %s",
		glog.PadRight("SUM", wHost, ' '),
		glog.PadRight("", wPid, ' '),
		glog.PadRight("", wCmd, ' '),
		glog.PadRight(glog.ProgressBar(cpu/100.0, wP), wCpu, ' '),
		glog.PadRight(glog.ProgressBar(mem/100.0, wP), wMem-wMemB, ' '),
		glog.PadLeft(glog.HumanReadableBytesIEC(memB*1024), wMemB, ' '),
	)
	if numHosts > 0 {
		log.Blank(
			glog.Bold()+"\033[97;40m%s %s %s"+glog.Reset()+" %s %s %s",
			glog.PadRight("AVG", wHost, ' '),
			glog.PadRight("", wPid, ' '),
			glog.PadRight("", wCmd, ' '),
			glog.PadRight(glog.ProgressBar(cpu/float64(numHosts)/100.0, wP), wCpu, ' '),
			glog.PadRight(glog.ProgressBar(mem/float64(numHosts)/100.0, wP), wMem-wMemB, ' '),
			glog.PadLeft(glog.HumanReadableBytesIEC(memB/float64(numHosts)*1024), wMemB, ' '),
		)
	}
}

type ProcessInfo struct {
	Timestamp time.Time
	Host      string
	PID       string
	CPU       string
	MEM       string
	VSZ       string
	RSS       string
	CMD       string
}

func NewProcessInfo(host, pid, cpu, mem, vsz, rss, cmd string) *ProcessInfo {
	return &ProcessInfo{
		Timestamp: time.Now(),
		Host:      host,
		PID:       pid,
		CPU:       cpu,
		MEM:       mem,
		VSZ:       vsz,
		RSS:       rss,
		CMD:       cmd,
	}
}

func MonitorRemoteProcesses(remoteHosts, processNames []string, dUpdate time.Duration) {
	list := []*ProcessInfo{}
	for _, remoteHost := range remoteHosts {
		psCmd := fmt.Sprintf("ps --no-headers -eo pid,pcpu,pmem,vsz,rss,cmd | grep -E '%s'", strings.Join(processNames, "|"))
		cmd := exec.Command("ssh", remoteHost, "-tt", psCmd)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Error("Error running SSH command: %s", glog.Error(err))
			return
		}

		seenPIDs := make(map[string]bool)
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)

			if len(fields) >= 6 {
				pid := fields[0]
				cpu := fields[1]
				mem := fields[2]
				vsz := fields[3]
				rss := fields[4]
				cmd := fields[5]

				for _, name := range processNames {
					re := regexp.MustCompile("\\b" + regexp.QuoteMeta(name) + "\\b")
					if re.MatchString(cmd) && !seenPIDs[pid] {
						list = append(list, NewProcessInfo(remoteHost, pid, cpu, mem, vsz, rss, cmd))
						seenPIDs[pid] = true
						break
					}
				}
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		var (
			hostA, hostB = list[i].Host, list[j].Host
			cmdA, cmdB   = list[i].CMD, list[j].CMD
			pidA, pidB   = list[i].PID, list[j].PID
		)
		if hostA != hostB {
			return hostA < hostB
		}
		if cmdA == cmdB {
			return pidA < pidB
		}
		return cmdA < cmdB
	})
	cpuPctTotal, memPctTotal, memKBytesTotal := 0.0, 0.0, 0.0

	fmt.Print("\033[2J\033[H")
	log.Blank("%s [ every %s ] %s\n", glog.Time(time.Now()), glog.Auto(dUpdate), glog.Auto(processNames))
	printHeader("HOST", "PID", "CMD", "CPU", "MEM", WHOST, WPID, WCMD, WCPU_HEAD, WMEM_HEAD, WMEMB)
	for _, p := range list {
		cpuPct, _ := glog.GetFloat(p.CPU)
		memPct, _ := glog.GetFloat(p.MEM)
		memKBytes, _ := glog.GetFloat(p.RSS)
		cpuPctTotal += cpuPct
		memPctTotal += memPct
		memKBytesTotal += memKBytes
		printRow(p.Host, p.PID, p.CMD, cpuPct, memPct, memKBytes, WHOST, WPID, WCMD, WCPU, WMEM, WMEMB, WP)
	}
	printFooter(cpuPctTotal, memPctTotal, memKBytesTotal, WHOST, WPID, WCMD, WCPU, WMEM, WMEMB, WP, len(remoteHosts))
}

func handleCtrlC() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Print("\033[H\033[2J")
		os.Exit(0)
	}()
}

func init() {
	glog.LoggerConfig.ShowDateTime = false
	glog.LoggerConfig.ShowRuntimeMilliseconds = false
	glog.LoggerConfig.ShowSubsystem = false
	glog.LoggerConfig.ShowIndicator = false
}

func main() {
	handleCtrlC()

	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s [update_freq] [remote_host] [process_1] ... <process_n>\n", glog.Auto(os.Args[0]))
		fmt.Printf("   or  %s [update_freq] [remote_host_1,remote_host_2,...,remote_host_n] [process_1] ... <process_n>\n", glog.Auto(os.Args[0]))
		os.Exit(1)
	}

	updateFreq, err := time.ParseDuration(os.Args[1])
	if err != nil {
		log.Error("Could not parse update frequency: %s", glog.Error(err))
		os.Exit(2)
	}

	fmt.Print("\033[H\033[2J")
	t := time.Now()
	for {
		MonitorRemoteProcesses(strings.Split(os.Args[2], ","), os.Args[3:], updateFreq)
		time.Sleep(updateFreq - time.Since(t))
		t = time.Now()
	}
}
