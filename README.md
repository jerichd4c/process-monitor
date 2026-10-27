<!-- Improved compatibility of back to top link: See: https://github.com/othneildrew/Best-README-Template/pull/73 -->
<a id="readme-top"></a>

<!-- PROJECT LOGO -->
<div align="center">
  <a href="https://github.com/jerichd4c/process-monitor">
    <img src="https://raw.githubusercontent.com/jerichd4c/logos/main/go_logo.svg" alt="Logo" width="80" height="80">
  </a>
</div>

<div align="center">

<h3 align="center">Process Monitor</h3>

  <p align="center">
    A full-screen terminal process monitor for Linux, in the spirit of top and htop, written in Go. Project for the Operating Systems course at URU.
  </p>
</div>



<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#features">Features</a></li>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li>
      <a href="#usage">Usage</a>
      <ul>
        <li><a href="#screen-layout">Screen Layout</a></li>
        <li><a href="#key-bindings">Key Bindings</a></li>
        <li><a href="#colors">Colors</a></li>
        <li><a href="#make-targets">Make Targets</a></li>
      </ul>
    </li>
    <li>
      <a href="#architecture">Architecture</a>
      <ul>
        <li><a href="#project-structure">Project Structure</a></li>
        <li><a href="#design-rules">Design Rules</a></li>
      </ul>
    </li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#known-limitations">Known Limitations</a></li>
    <li><a href="#acknowledgments">Acknowledgments</a></li>
  </ol>
</details>



<!-- ABOUT THE PROJECT -->
## About The Project

Process Monitor shows the state of a Linux machine and lets you act on its processes: terminate them, suspend and resume them, and change their priority.

Everything is read directly from `/proc`, the interface the kernel exposes about its processes. The program never runs `ps`, `top` or any other external program: it computes the CPU percentage, the totals and the summary on its own.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Features

* **Process table** with PID, parent PID, user, state, %CPU, %MEM, resident memory, thread count, priority, nice value, CPU time and full command line.
* **Summary** of total, running, sleeping, stopped and zombie processes, plus CPU usage, load average and memory.
* **%CPU computed from deltas:** the CPU time a process used between two samples divided by the CPU time of the whole machine over the same interval.
* **Sorting** by CPU, memory or PID, and a **name filter**.
* **Process tree** (parent to child) with branch lines.
* **Thread view** with the state and %CPU of every thread of a process.
* **CPU history graph** of a process, auto-scaled, with the last 300 samples.
* **Signals:** terminate (SIGTERM), kill (SIGKILL), suspend (SIGSTOP) and resume (SIGCONT), each with the confirmation it needs.
* **Priority:** show and change the nice value of a process.
* **Highlighting** of zombies and processes blocked on I/O.
* Adapts to the terminal size, redraws without flicker, restores the terminal on exit and has a configurable refresh interval (0.5 s to 10 s).
* Clear messages instead of crashes when a process no longer exists or permission is missing.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built With

* [![Go][Go-badge]][Go-url]
* [![tcell][tcell-badge]][tcell-url]

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- GETTING STARTED -->
## Getting Started

### Prerequisites

* Linux. On Windows, use Ubuntu on WSL2 and run everything from the Ubuntu terminal, never from PowerShell: `/proc` does not exist on Windows.
* Go 1.25 or higher, git and make:
  ```sh
  sudo apt update && sudo apt install -y golang-go git make
  ```
* An xterm-like terminal, such as Windows Terminal.

### Installation

1. Clone the repository
   ```sh
   git clone https://github.com/jerichd4c/process-monitor.git
   ```
2. Navigate to the project directory
   ```sh
   cd process-monitor
   ```
3. Build the binary (it is created as `./process-monitor`)
   ```sh
   make
   ```
4. Run it
   ```sh
   ./process-monitor
   ```

Quit with `q` or `Ctrl+C`; the terminal is left exactly as it was. Run it with `sudo` to act on processes of other users and to raise priorities: the code is the same, only what the kernel allows changes.

The Makefile builds with `-tags tcell_minimal` (see [Design Rules](#design-rules)). The equivalent manual command is `go build -tags tcell_minimal -o process-monitor .`.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- USAGE EXAMPLES -->
## Usage

### Screen Layout

The screen has three fixed zones: the summary on top, the table in the middle and a status bar at the bottom. Press `?` inside the program to see every key.

```
Tasks: 36 total, 1 running, 34 sleeping, 0 stopped, 1 zombie          <- summary
CPU:   6.4%   Load average: 0.42 0.30 0.12
Mem: 6.7G total, 652.4M used, 6.1G available
    PID    PPID USER     S  %CPU  %MEM     RES  THR  PR  NI      TIME  COMMAND   <- header
    534     338 sebas    R   6.3   0.1    7.2M    1  20   0   0:02.07  yes       <- selected row
      1       0 root     S   0.0   0.2   15.1M    1  20   0   0:01.00  /sbin/init
    ...
 ?:help  q:quit   sort[CPU]  refresh[1.0s]                             <- status bar
```

In `%CPU` and `%MEM`, 100 means the whole machine (all CPUs). Memory in use is total minus available. Kernel threads are not shown. States are `R` running, `S` sleeping, `D` waiting on I/O, `T` stopped, `Z` zombie and `I` idle.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Key Bindings

| Group | Key | Action |
|---|---|---|
| Move | Up / Down | Select the previous or next row |
| | PgUp / PgDn | Move one screen |
| | Home / End | Go to the first or last process |
| Sort | `c` | By CPU, highest first (initial order) |
| | `m` | By memory |
| | `p` | By PID |
| | `t` | Toggle the parent-to-child tree |
| Filter | `/` | Type part of a name. Enter keeps it, Esc clears it |
| Views | `h` | Threads of the selected process |
| | `g` | CPU history graph of the selected process |
| Actions | `k` | Terminate (SIGTERM), asks `[y/N]` |
| | `K` | Kill (SIGKILL), requires typing `yes` and Enter |
| | `s` | Suspend (SIGSTOP) |
| | `r` | Resume (SIGCONT) |
| | `n` | Change priority: type a nice value from -20 to 19 and Enter |
| Refresh | `+` / `-` | Slower (up to 10 s) / faster (down to 0.5 s) |
| General | `?` | Help with every key |
| | Esc | Go back: closes a view, cancels a question or clears the filter. Never quits |
| | `q` / `Ctrl+C` | Quit |

Actions apply to the selected row, and the selection follows the process rather than the position, so it stays on the same process when the table reorders. Inside the thread, graph and help views the action keys do nothing.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Colors

| Where | Color | Meaning |
|---|---|---|
| Row | Reversed | The selected process |
| Row | Red, bold | Zombie: it ended and its parent has not collected it |
| Row | Yellow, bold | Blocked waiting for input or output (`D`) |
| Row | Teal | Stopped (`T`) |
| Bottom bar | Yellow | A question is waiting for an answer |
| Bottom bar | Green | The action was applied |
| Bottom bar | Red | The action could not be applied, with the reason |

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Make Targets

| Command | Description |
|---|---|
| `make` | Build `./process-monitor` |
| `make run` | Build and start the monitor |
| `make test` | Run the automated tests |
| `make race` | Run the tests with Go's race detector |
| `make cover` | Run the tests and show the coverage |
| `make deps` | Check the two dependency rules described below |
| `make check` | Format check, `go vet`, dependency rules and tests: everything that must pass before a delivery |
| `make clean` | Remove the binary |
| `make help` | List the targets |

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- ARCHITECTURE -->
## Architecture

The code is split in four packages and each one has a single responsibility: `proc` reads, `monitor` calculates, `control` acts and `ui` shows. Data flows one way: the kernel exposes it in `/proc`, `proc` reads it, `monitor` turns it into percentages and totals, and `ui` draws it. Actions flow the opposite way: `ui` receives the key and `control` makes the system call, after checking through `proc` that the target is still the same process.

### Project Structure

```
process-monitor/
├── main.go            Entry point: only calls ui.Run()
├── Makefile           Build, test and verification targets
├── go.mod, go.sum     Module name and dependencies
├── proc/              READ: raw data from /proc (process, thread, system, memory, users)
├── monitor/           CALCULATE: %CPU, summary, sort and filter, tree, history
├── control/           ACT: signals and priority
└── ui/                SHOW: screen and keyboard
```

The sampling lives in `monitor/sampler.go`. On every refresh it reads the machine-wide CPU time, memory and load; lists the PIDs; reads `stat`, `status` and `cmdline` for each one (skipping the ones that disappeared); drops kernel threads; computes the %CPU against the previous reading, keyed by PID and start time; counts states for the summary; and records the value in the process history. The UI then filters, sorts or builds the tree, keeps the selection on the same PID and draws.

Only the main loop reads or modifies the interface state. The goroutine that waits for keys just passes events through a channel, so no locks are needed and the race detector reports nothing.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Design Rules

* **Standard library only in the core.** `proc`, `monitor`, `control` and `main.go` use only Go's standard library, and system calls go through the `syscall` package. Only `ui` uses an external library (tcell).
* **The binary cannot run other programs.** It is built with `-tags tcell_minimal`, so tcell's code that could execute `infocmp` for unknown terminals is left out and `os/exec` is not linked.
* `make deps` checks both rules and fails if either is broken.
* **Esc never quits.** It only goes back, so the program is not closed by pressing it once too many.
* **Safe actions.** Suspend and resume are reversible and need no confirmation; terminate asks `y`, kill requires typing `yes`. The target is fixed when the key is pressed, not when the answer is given. The program refuses to act on itself, and checks the process start time right before sending a signal so it does not hit a different process that reused the PID.
* Process names are stripped of control characters and invalid UTF-8 before being drawn.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- ROADMAP -->
## Roadmap

The monitor already covers its core features; these are ideas to improve the interface. Any of them has to respect the [design rules](#design-rules): the data and control packages stay standard-library only, so these changes live in `ui`, or are plain calculations in `monitor`.

- [ ] Reverse the sort direction and sort by more columns (user, threads, CPU time...). Today the table can only be ordered by CPU, memory or PID.
- [ ] Show CPU and memory as color-coded bar meters in the summary, with an optional bar per CPU core, instead of plain numbers.
- [ ] Add a detail view for the selected process with its full command line, which is cut at the screen edge in the table.
- [ ] Collapse and expand branches in the process tree.
- [ ] Add mouse support: select a row, click a column header to sort and scroll with the wheel.
- [ ] Choose which columns are shown, and remember the sort order and refresh interval between runs.
- [ ] Draw memory history in the graph view next to the CPU history.
- [ ] Investigate supporting more terminal types (`TERM=linux`, `screen-256color`) without linking `os/exec` back into the binary.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- KNOWN LIMITATIONS -->
## Known Limitations

* **Linux only:** it depends on `/proc` and Unix signals.
* **xterm-like terminals only:** because of `tcell_minimal`, only the xterm, tmux, vt100, vt220 and ansi families work. The plain text console (`TERM=linux`) exits with a terminal type unsupported error. Windows Terminal with WSL uses `xterm-256color`, which is supported.
* **Kernel threads in WSL2:** they are not visible there, so the filter that hides them was only verified with unit tests.
* **Clock ticks per second are assumed to be 100,** the usual Linux value, because Go cannot query `sysconf` without cgo.
* **User names come from `/etc/passwd`;** a user that is not listed there is shown by its number.
* **History is limited to 300 samples per process:** 5 minutes at the 1 s interval.
* **Permissions:** without root you cannot raise priorities or act on processes of other users; the program says so with a message. A stopped process does not react to SIGTERM until it is resumed, which is kernel behavior.
* The `D` state lasts milliseconds, so it is hard to see highlighted on purpose.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- ACKNOWLEDGMENTS -->
## Acknowledgments

* [tcell](https://github.com/gdamore/tcell) for the terminal screen and keyboard handling

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- MARKDOWN LINKS & IMAGES -->
[Go-badge]: https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white
[Go-url]: https://go.dev/
[tcell-badge]: https://img.shields.io/badge/tcell-2F2F2F?style=for-the-badge
[tcell-url]: https://github.com/gdamore/tcell
