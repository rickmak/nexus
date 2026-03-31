package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/nexus/nexus/packages/workspace-daemon/pkg/runtime"
	"github.com/nexus/nexus/packages/workspace-daemon/pkg/runtime/dind"
	"github.com/nexus/nexus/packages/workspace-daemon/pkg/runtime/lxc"
	"github.com/nexus/nexus/packages/workspace-daemon/pkg/server"
)

type CommandRunner struct{}

func (r *CommandRunner) Run(ctx context.Context, dir string, cmd string, args ...string) error {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = dir
	return c.Run()
}

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	workspaceDir := flag.String("workspace-dir", "/workspace", "Workspace directory path")
	token := flag.String("token", "", "JWT secret token for authentication")
	flag.Parse()

	if *token == "" {
		log.Fatal("Error: --token is required")
	}

	if err := runServer(*port, *workspaceDir, *token); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runServer(port int, workspaceDir string, token string) error {
	srv, err := server.NewServer(port, workspaceDir, token)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	runner := &CommandRunner{}
	dindDriver := dind.NewDriver(runner)
	lxcDriver := lxc.NewDriver(runner)

	_, dockerErr := exec.LookPath("docker")
	dindAvailable := dockerErr == nil

	_, lxcErr := exec.LookPath("lxc")
	lxcAvailable := lxcErr == nil

	_, codexErr := exec.LookPath("codex")
	codexAvailable := codexErr == nil

	_, opencodeErr := exec.LookPath("opencode")
	opencodeAvailable := opencodeErr == nil

	capabilities := []runtime.Capability{
		{Name: "runtime.dind", Available: dindAvailable},
		{Name: "runtime.lxc", Available: lxcAvailable},
		{Name: "spotlight.tunnel", Available: true},
		{Name: "auth.profile.git", Available: true},
		{Name: "auth.profile.codex", Available: codexAvailable},
		{Name: "auth.profile.opencode", Available: opencodeAvailable},
	}

	drivers := map[string]runtime.Driver{
		"dind": dindDriver,
		"lxc":  lxcDriver,
	}

	factory := runtime.NewFactory(capabilities, drivers)
	srv.SetRuntimeFactory(factory)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		srv.Shutdown()
	}()

	log.Printf("Workspace daemon started on port %d", port)
	return srv.Start()
}
