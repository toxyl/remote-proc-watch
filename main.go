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

var log = glog.NewLoggerSimple("rpw")

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
			hostA = list[i].Host
			hostB = list[j].Host
			cmdA  = list[i].CMD
			cmdB  = list[j].CMD
			pidA  = list[i].PID
			pidB  = list[j].PID
		)
		if hostA != hostB {
			return hostA < hostB
		}
		if cmdA == cmdB {
			return pidA < pidB
		}
		return cmdA < cmdB
	})
	cpuPctTotal := 0.0
	memPctTotal := 0.0
	memKBytesTotal := 0.0

	const (
		WP        = 20
		WHOST     = 32
		WPID      = 10
		WCMD      = 35
		WCPU      = 55
		WCPU_HEAD = WCPU - 18
		WMEM      = 65
		WMEM_HEAD = WMEM - 18
		WMEMB     = 15
	)

	fmt.Print(glog.StoreCursor())
	log.Blank(
		"%s [ every %s ] %s\n",
		glog.Time(time.Now()),
		glog.Auto(dUpdate),
		glog.Auto(processNames),
	)
	log.Blank(
		glog.Bold()+"\033[97;40m%s %s %s %s %s"+glog.Reset(),
		glog.PadRight("HOST", WHOST, ' '),
		glog.PadRight("PID", WPID, ' '),
		glog.PadRight("CMD", WCMD, ' '),
		glog.PadRight("CPU", WCPU_HEAD, ' '),
		glog.PadRight("MEM", WMEM_HEAD, ' '),
	)
	for _, p := range list {
		cpuPct, _ := glog.GetFloat(p.CPU)
		memPct, _ := glog.GetFloat(p.MEM)
		memKBytes, _ := glog.GetFloat(p.RSS)
		cpuPctTotal += cpuPct
		memPctTotal += memPct
		memKBytesTotal += memKBytes
		log.Blank(
			"%s %s %s %s   %s %s",
			glog.PadRight(glog.Auto(p.Host), WHOST, ' '),
			glog.PadRight(glog.Auto(p.PID), WPID, ' '),
			glog.PadRight(glog.Auto(p.CMD), WCMD, ' '),
			glog.PadRight(glog.ProgressBar(cpuPct/100.0, WP), WCPU, ' '),
			glog.PadRight(glog.ProgressBar(memPct/100.0, WP), WMEM-WMEMB, ' '),
			glog.PadLeft(glog.HumanReadableBytesIEC(memKBytes*1024), WMEMB, ' '),
		)
	}
	log.Blank(
		"\033[97;40m%s %s "+glog.Bold()+"%s"+glog.Reset()+" %s   %s %s",
		glog.PadRight("", WHOST, ' '),
		glog.PadRight("", WPID, ' '),
		glog.PadRight("TOTAL", WCMD, ' '),
		glog.PadRight(glog.ProgressBar(cpuPctTotal/100.0, WP), WCPU, ' '),
		glog.PadRight(glog.ProgressBar(memPctTotal/100.0, WP), WMEM-WMEMB, ' '),
		glog.PadLeft(glog.HumanReadableBytesIEC(memKBytesTotal*1024), WMEMB, ' '),
	)
	fmt.Print(glog.RestoreCursor())
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

	remoteHosts := strings.Split(os.Args[2], ",")
	processNames := os.Args[3:]

	fmt.Print("\033[H\033[2J")
	t := time.Now()
	for {
		MonitorRemoteProcesses(remoteHosts, processNames, updateFreq)
		time.Sleep(updateFreq - time.Since(t))
		t = time.Now()
	}
}
