# remote-proc-watch
`remote-proc-watch` is a simple tool designed to monitor specific processes on one or more remote hosts using `ssh` and `ps`. It provides essential information such as PID, process name, CPU usage, and memory usage in both percentage and human-readable formatted bytes.

## Installation
```sh
CGO_ENABLED=0 go build -o /usr/local/bin/rpw .
```

## Usage
```sh
rpw [UPDATE_INTERVAL] [SSH_HOST] [PROCESS1] <PROCESS2> ...
# or
rpw [UPDATE_INTERVAL] [SSH_HOST_1,SSH_HOST_2,...,SSH_HOST_N] [PROCESS1] <PROCESS2> ...
```
- The tool relies on your SSH configuration to connect with hosts. Ensure you can connect to the host without user interaction.
- Processes are matched on word boundaries, allowing for exact matches regardless of their position in the command.

## Example
```sh
rpw 2s host1,host2 spider ossh
```
### Output (refreshed every 2s)
```
 17:37:09 [ every 2s ] spider, ossh

 HOST                             PID        CMD                                 CPU                                   MEM                                            
 host1                            802        /etc/ossh/ossh                      ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 0.90%            ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 2.30%           94.10 MiB
 host1                            1081       /usr/local/bin/spider               ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 0.20%            ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 0.30%           14.93 MiB
 host2                            288927     /etc/ossh/ossh                      ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 3.00%            ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 0.40%          147.62 MiB
 host2                            108758     /usr/local/bin/spider               ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 0.70%            ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 0.00%           23.21 MiB
                                             TOTAL                               ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 4.80%            ▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫▫ 3.00%          279.86 MiB
```
