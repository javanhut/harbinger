package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/javanhut/harbinger/internal/conflict"
	"github.com/javanhut/harbinger/internal/git"
	"github.com/javanhut/harbinger/internal/notify"
)

type Options struct {
	PollInterval time.Duration
}

type Monitor struct {
	repo             *git.Repository
	options          Options
	notifier         *notify.Notifier
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	lastRemoteCommit string
}

func New(repoPath string, options Options) (*Monitor, error) {
	repo, err := git.NewRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	notifier := notify.New()

	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		repo:     repo,
		options:  options,
		notifier: notifier,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (m *Monitor) Start() error {
	// Get initial state
	branch, err := m.repo.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if err := m.repo.Fetch(); err != nil {
		return fmt.Errorf("failed to fetch remote: %w", err)
	}

	remoteCommit, err := m.repo.GetRemoteCommit(branch)
	if err != nil {
		log.Printf("Warning: failed to get remote commit (branch might not have upstream): %v", err)
	} else {
		m.lastRemoteCommit = remoteCommit
	}

	m.wg.Add(1)
	go m.monitorLoop()

	return nil
}

func (m *Monitor) Stop() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.options.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.checkForChanges(); err != nil {
				log.Printf("Error checking for changes: %v", err)
			}
		}
	}
}

func (m *Monitor) checkForChanges() error {
	// Fetch latest changes
	if err := m.repo.Fetch(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	branch, err := m.repo.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if remote has changed
	remoteCommit, err := m.repo.GetRemoteCommit(branch)
	if err != nil {
		// Branch might not have upstream
		return nil
	}

	localCommit, err := m.repo.GetLocalCommit(branch)
	if err != nil {
		return fmt.Errorf("failed to get local commit: %w", err)
	}

	// Check if remote has new changes
	if remoteCommit != m.lastRemoteCommit && m.lastRemoteCommit != "" {
		m.notifier.NotifyRemoteChange(branch, remoteCommit)

		// Check for potential conflicts
		conflicts, err := m.repo.CheckForConflicts(fmt.Sprintf("origin/%s", branch))
		if err != nil {
			log.Printf("Error checking for conflicts: %v", err)
		} else if len(conflicts) > 0 {
			m.handleConflicts(conflicts)
		}
	}

	// Check if local is behind remote
	if localCommit != remoteCommit {
		m.notifier.NotifyOutOfSync(branch, localCommit, remoteCommit)
	}

	m.lastRemoteCommit = remoteCommit
	return nil
}

func (m *Monitor) handleConflicts(conflicts []git.Conflict) {
	m.notifier.NotifyConflicts(len(conflicts))

	// Launch conflict resolution UI
	resolver := conflict.NewResolver(m.repo)
	if err := resolver.ResolveConflicts(conflicts); err != nil {
		log.Printf("Error resolving conflicts: %v", err)
	}
}
