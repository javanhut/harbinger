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
	"github.com/javanhut/harbinger/pkg/config"
)

type Options struct {
	PollInterval time.Duration
}

type Monitor struct {
	repo             *git.Repository
	options          Options
	notifier         *notify.Notifier
	config           *config.Config
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	lastRemoteCommit string
	lastSyncStatus   bool // Track if we were in sync last time
	currentBranch    string
}

func New(repoPath string, options Options) (*Monitor, error) {
	repo, err := git.NewRepository(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	notifier := notify.New()

	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		repo:     repo,
		options:  options,
		notifier: notifier,
		config:   cfg,
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

	// Check if we've switched branches
	if m.currentBranch != "" && m.currentBranch != branch {
		log.Printf("Switched from branch '%s' to '%s'", m.currentBranch, branch)
		m.lastRemoteCommit = "" // Reset tracking
		m.lastSyncStatus = false
	}
	m.currentBranch = branch

	// Check sync status
	inSync, err := m.repo.IsInSync(branch)
	if err != nil {
		// Branch might not have upstream
		log.Printf("Warning: unable to check sync status: %v", err)
		return nil
	}

	// If we just became in sync, notify with green checkmark
	if inSync && !m.lastSyncStatus {
		m.notifier.NotifyInSync(branch)
	}

	// Check if we're behind remote
	isBehind, behindCount, err := m.repo.IsBehindRemote(branch)
	if err != nil {
		log.Printf("Warning: unable to check if behind remote: %v", err)
	} else if isBehind {
		m.notifier.NotifyBehindRemote(branch, behindCount)

		// Auto-pull if enabled and no uncommitted changes
		if m.config.AutoPull {
			if err := m.attemptAutoPull(branch, behindCount); err != nil {
				log.Printf("Auto-pull failed: %v", err)
			}
		}
	}

	// Check for conflicts if we're not in sync
	if !inSync {
		conflicts, err := m.repo.CheckForConflicts(fmt.Sprintf("origin/%s", branch))
		if err != nil {
			log.Printf("Error checking for conflicts: %v", err)
		} else if len(conflicts) > 0 {
			m.handleConflicts(conflicts)
		}
	}

	m.lastSyncStatus = inSync
	return nil
}

func (m *Monitor) attemptAutoPull(branch string, commitCount int) error {
	// Check if we have uncommitted changes
	hasChanges, err := m.repo.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if hasChanges {
		log.Printf("Cannot auto-pull: uncommitted changes in working directory")
		return fmt.Errorf("uncommitted changes prevent auto-pull")
	}

	// Attempt to pull
	log.Printf("Auto-pulling %d commit(s) into branch '%s'", commitCount, branch)
	if err := m.repo.Pull(); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	m.notifier.NotifyAutoPull(branch, commitCount)
	log.Printf("Successfully auto-pulled %d commit(s)", commitCount)
	return nil
}

func (m *Monitor) handleConflicts(conflicts []git.Conflict) {
	m.notifier.NotifyConflicts(len(conflicts))

	// Only launch conflict resolution UI if auto_resolve is enabled
	if m.config.AutoResolve {
		log.Println("Auto-resolving conflicts (use 'harbinger resolve' to manually resolve)")
		resolver := conflict.NewResolver(m.repo)
		if err := resolver.ResolveConflicts(conflicts); err != nil {
			log.Printf("Error resolving conflicts: %v", err)
		}
	} else {
		log.Println("Conflicts detected. Use 'harbinger resolve' to manually resolve them.")
	}
}
