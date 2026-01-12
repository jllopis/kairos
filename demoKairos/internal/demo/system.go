package demo

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FlowSpec describes the demo flow for logging purposes.
type FlowSpec struct {
	Description string
}

// Flow creates a flow spec with a human-readable description.
func Flow(description string) FlowSpec {
	return FlowSpec{Description: description}
}

// AgentSpec defines how to launch a demo agent process.
type AgentSpec struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	WorkDir string
}

type runningAgent struct {
	spec AgentSpec
	cmd  *exec.Cmd
}

// System is a fluent builder for the demo entrypoint.
type System struct {
	root   string
	agents []AgentSpec
	flow   FlowSpec
	env    map[string]string
	logger *log.Logger
}

// NewSystem builds a demo system rooted at the demoKairos module.
func NewSystem() (*System, error) {
	root, err := ResolveDemoRoot()
	if err != nil {
		return nil, err
	}
	return &System{
		root:   root,
		env:    map[string]string{},
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}, nil
}

// Root returns the resolved demo root path.
func (s *System) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

// SetRoot overrides the demo root path after validating it.
func (s *System) SetRoot(root string) error {
	if s == nil {
		return errors.New("demo system is nil")
	}
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return errors.New("demo root is empty")
	}
	if !isDemoRoot(trimmed) {
		return fmt.Errorf("invalid demo root: %s", trimmed)
	}
	s.root = trimmed
	return nil
}

// WithAgent registers an agent process to run as part of the demo.
func (s *System) WithAgent(agent AgentSpec) *System {
	if s == nil {
		return s
	}
	s.agents = append(s.agents, agent)
	return s
}

// WithFlow stores a flow description for demo logs.
func (s *System) WithFlow(flow FlowSpec) *System {
	if s == nil {
		return s
	}
	s.flow = flow
	return s
}

// WithEnv sets a shared environment variable for all agents.
func (s *System) WithEnv(key, value string) *System {
	if s == nil {
		return s
	}
	if s.env == nil {
		s.env = map[string]string{}
	}
	if strings.TrimSpace(key) != "" {
		s.env[key] = value
	}
	return s
}

// WithEnvMap merges shared environment variables for all agents.
func (s *System) WithEnvMap(values map[string]string) *System {
	if s == nil {
		return s
	}
	for key, value := range values {
		s.WithEnv(key, value)
	}
	return s
}

// Run starts all agents and waits until the context is canceled or one exits.
func (s *System) Run(ctx context.Context) error {
	if s == nil {
		return errors.New("demo system is nil")
	}
	if len(s.agents) == 0 {
		return errors.New("demo system has no agents")
	}
	if strings.TrimSpace(s.flow.Description) != "" {
		s.logger.Printf("Flow: %s", s.flow.Description)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	running := make([]*runningAgent, 0, len(s.agents))
	errCh := make(chan error, len(s.agents))
	var wg sync.WaitGroup
	var firstErr error

	for _, agent := range s.agents {
		cmd := exec.Command(agent.Command, agent.Args...)
		if agent.WorkDir != "" {
			cmd.Dir = agent.WorkDir
		} else if s.root != "" {
			cmd.Dir = s.root
		}
		cmd.Env = mergeEnv(os.Environ(), s.env, agent.Env)
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start %s: %w", agent.Name, err)
		}
		s.logger.Printf("Started %s (pid=%d)", agent.Name, cmd.Process.Pid)
		running = append(running, &runningAgent{spec: agent, cmd: cmd})

		if stdout != nil {
			go s.pipeOutput(agent.Name, stdout)
		}
		if stderr != nil {
			go s.pipeOutput(agent.Name, stderr)
		}

		wg.Add(1)
		go func(name string, cmd *exec.Cmd) {
			defer wg.Done()
			err := cmd.Wait()
			if ctx.Err() != nil {
				return
			}
			if err == nil {
				err = fmt.Errorf("%s exited", name)
			}
			errCh <- err
		}(agent.Name, cmd)
	}

	select {
	case <-ctx.Done():
		// shutdown requested
	case err := <-errCh:
		firstErr = err
		s.logger.Printf("Stopping after agent exit: %v", err)
		cancel()
	}

	s.stopAll(running)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (s *System) stopAll(running []*runningAgent) {
	for _, agent := range running {
		if agent.cmd.Process == nil {
			continue
		}
		_ = agent.cmd.Process.Signal(os.Interrupt)
	}
	// Force-stop after a short grace period.
	time.AfterFunc(2*time.Second, func() {
		for _, agent := range running {
			if agent.cmd.Process == nil {
				continue
			}
			_ = agent.cmd.Process.Kill()
		}
	})
}

func (s *System) pipeOutput(name string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		s.logger.Printf("[%s] %s", name, line)
	}
}

func mergeEnv(base []string, maps ...map[string]string) []string {
	merged := map[string]string{}
	for _, pair := range base {
		if pair == "" {
			continue
		}
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		merged[key] = value
	}
	for _, m := range maps {
		for key, value := range m {
			if strings.TrimSpace(key) == "" {
				continue
			}
			if value == "" {
				continue
			}
			merged[key] = value
		}
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s=%s", key, merged[key]))
	}
	return out
}

// ResolveDemoRoot locates the demoKairos root directory.
func ResolveDemoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if isDemoRoot(cwd) {
		return cwd, nil
	}
	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if isDemoRoot(dir) {
			return dir, nil
		}
	}
	candidate := filepath.Join(cwd, "demoKairos")
	if isDemoRoot(candidate) {
		return candidate, nil
	}
	return "", fmt.Errorf("demo root not found from %s", cwd)
}

func isDemoRoot(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	for _, required := range []string{"cmd", "data", "go.mod"} {
		if stat, err := os.Stat(filepath.Join(path, required)); err != nil || stat == nil {
			return false
		}
	}
	return true
}
