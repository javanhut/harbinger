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
	RemoteBranch string // Optional: specific remote branch to monitor
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
	targetBranch     string // The remote branch we're monitoring
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
		repo:         repo,
		options:      options,
		notifier:     notifier,
		config:       cfg,
		ctx:          ctx,
		cancel:       cancel,
		targetBranch: options.RemoteBranch,
	}, nil
}

func (m *Monitor) Start() error {
	// Get initial state
	branch, err := m.repo.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	m.currentBranch = branch

	log.Printf("[%s] Starting monitor for repository: %s", time.Now().Format(time.RFC3339), m.repo.Path())
	log.Printf("[%s] Current branch: %s", time.Now().Format(time.RFC3339), branch)
	if m.targetBranch != "" {
		log.Printf("[%s] Monitoring remote branch: %s", time.Now().Format(time.RFC3339), m.targetBranch)
	}
	log.Printf("[%s] Poll interval: %s", time.Now().Format(time.RFC3339), m.options.PollInterval)

	if err := m.repo.Fetch(); err != nil {
		return fmt.Errorf("failed to fetch remote: %w", err)
	}

	// Determine which branch to compare against
	compareBranch := branch
	if m.targetBranch != "" {
		compareBranch = m.targetBranch
	}

	remoteCommit, err := m.repo.GetRemoteCommit(compareBranch)
	if err != nil {
		log.Printf("[%s] Warning: failed to get remote commit (branch might not have upstream): %v", time.Now().Format(time.RFC3339), err)
	} else {
		m.lastRemoteCommit = remoteCommit
		log.Printf("[%s] Remote HEAD (%s): %s", time.Now().Format(time.RFC3339), compareBranch, remoteCommit[:8])
	}

	// Check initial sync status
	if m.targetBranch != "" {
		// Get local commit for comparison
		localCommit, _ := m.repo.GetLocalCommit(branch)
		m.lastSyncStatus = localCommit == remoteCommit
		if m.lastSyncStatus {
			log.Printf("[%s] Status: In sync with remote branch %s", time.Now().Format(time.RFC3339), compareBranch)
		} else {
			log.Printf("[%s] Status: Not in sync with remote branch %s", time.Now().Format(time.RFC3339), compareBranch)
		}
	} else {
		inSync, err := m.repo.IsInSync(branch)
		if err != nil {
			log.Printf("[%s] Warning: unable to check initial sync status: %v", time.Now().Format(time.RFC3339), err)
		} else {
			m.lastSyncStatus = inSync
			if inSync {
				log.Printf("[%s] Status: In sync with remote", time.Now().Format(time.RFC3339))
			} else {
				log.Printf("[%s] Status: Not in sync with remote", time.Now().Format(time.RFC3339))
			}
		}
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
	log.Printf("[%s] Checking for changes...", time.Now().Format(time.RFC3339))
	
	// Fetch latest changes
	if err := m.repo.Fetch(); err != nil {
		log.Printf("[%s] Error: Failed to fetch remote changes: %v", time.Now().Format(time.RFC3339), err)
		return fmt.Errorf("failed to fetch: %w", err)
	}

	branch, err := m.repo.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if we've switched branches
	if m.currentBranch != "" && m.currentBranch != branch {
		log.Printf("[%s] Branch switch detected: '%s' -> '%s'", time.Now().Format(time.RFC3339), m.currentBranch, branch)
		m.lastRemoteCommit = "" // Reset tracking
		m.lastSyncStatus = false
	}
	m.currentBranch = branch

	// Determine which branch to compare against
	compareBranch := branch
	if m.targetBranch != "" {
		compareBranch = m.targetBranch
		log.Printf("[%s] Comparing current branch '%s' against remote branch '%s'", time.Now().Format(time.RFC3339), branch, compareBranch)
	}

	// Get current commits
	localCommit, err := m.repo.GetLocalCommit(branch)
	if err != nil {
		log.Printf("[%s] Warning: unable to get local commit: %v", time.Now().Format(time.RFC3339), err)
	} else {
		log.Printf("[%s] Local HEAD: %s", time.Now().Format(time.RFC3339), localCommit[:8])
	}

	remoteCommit, err := m.repo.GetRemoteCommit(compareBranch)
	if err != nil {
		log.Printf("[%s] Warning: unable to get remote commit: %v", time.Now().Format(time.RFC3339), err)
	} else {
		log.Printf("[%s] Remote HEAD (%s): %s", time.Now().Format(time.RFC3339), compareBranch, remoteCommit[:8])
	}

	// Check sync status - when monitoring a different branch, we check if local matches that remote branch
	var inSync bool
	if m.targetBranch != "" {
		// Compare local branch against specified remote branch
		inSync = localCommit == remoteCommit
	} else {
		// Normal sync check against same branch
		inSync, err = m.repo.IsInSync(branch)
		if err != nil {
			// Branch might not have upstream
			log.Printf("[%s] Warning: unable to check sync status: %v", time.Now().Format(time.RFC3339), err)
			return nil
		}
	}

	// Log sync status
	if inSync {
		log.Printf("[%s] Status: In sync with remote", time.Now().Format(time.RFC3339))
	} else {
		log.Printf("[%s] Status: Not in sync with remote", time.Now().Format(time.RFC3339))
	}

	// If we just became in sync, notify with green checkmark
	if inSync && !m.lastSyncStatus {
		log.Printf("[%s] Branch is now in sync! Sending notification.", time.Now().Format(time.RFC3339))
		m.notifier.NotifyInSync(branch)
	}

	// Auto-resolve when out of sync (if enabled)
	if !inSync && m.config.AutoResolve {
		log.Printf("[%s] Auto-resolve is enabled, attempting to sync with %s...", time.Now().Format(time.RFC3339), compareBranch)
		if err := m.attemptAutoResolve(branch, compareBranch); err != nil {
			log.Printf("[%s] Auto-resolve failed: %v", time.Now().Format(time.RFC3339), err)
		}
		// Re-check sync status after auto-resolve attempt
		if m.targetBranch != "" {
			localCommit, _ := m.repo.GetLocalCommit(branch)
			remoteCommit, _ := m.repo.GetRemoteCommit(compareBranch)
			inSync = localCommit == remoteCommit
		} else {
			inSync, _ = m.repo.IsInSync(branch)
		}
	}

	// Check if we're behind remote (only when monitoring same branch)
	if m.targetBranch == "" {
		isBehind, behindCount, err := m.repo.IsBehindRemote(branch)
		if err != nil {
			log.Printf("[%s] Warning: unable to check if behind remote: %v", time.Now().Format(time.RFC3339), err)
		} else if isBehind {
			log.Printf("[%s] Branch is %d commit(s) behind remote", time.Now().Format(time.RFC3339), behindCount)
			m.notifier.NotifyBehindRemote(branch, behindCount)

			// Auto-sync if enabled and no uncommitted changes  
			if m.config.AutoSync || m.config.AutoPull { // Support deprecated AutoPull for backward compatibility
				log.Printf("[%s] Auto-sync is enabled, attempting to pull changes...", time.Now().Format(time.RFC3339))
				if err := m.attemptAutoPull(branch, behindCount); err != nil {
					log.Printf("[%s] Auto-sync failed: %v", time.Now().Format(time.RFC3339), err)
				}
			}
		}
	}

	// Check for conflicts if we're not in sync
	if !inSync {
		log.Printf("[%s] Checking for potential conflicts...", time.Now().Format(time.RFC3339))
		conflicts, err := m.repo.CheckForConflicts(fmt.Sprintf("origin/%s", compareBranch))
		if err != nil {
			log.Printf("[%s] Error checking for conflicts: %v", time.Now().Format(time.RFC3339), err)
		} else if len(conflicts) > 0 {
			log.Printf("[%s] Found %d conflicting file(s) with %s", time.Now().Format(time.RFC3339), len(conflicts), compareBranch)
			m.handleConflicts(conflicts)
		} else {
			log.Printf("[%s] No conflicts detected with %s", time.Now().Format(time.RFC3339), compareBranch)
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

func (m *Monitor) attemptAutoResolve(currentBranch, remoteBranch string) error {
	// Check if we have uncommitted changes
	hasChanges, err := m.repo.HasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if hasChanges {
		log.Printf("[%s] Cannot auto-resolve: uncommitted changes in working directory", time.Now().Format(time.RFC3339))
		return fmt.Errorf("uncommitted changes prevent auto-resolve")
	}

	// Check for conflicts before attempting merge
	conflicts, err := m.repo.CheckForConflicts(fmt.Sprintf("origin/%s", remoteBranch))
	if err != nil {
		return fmt.Errorf("failed to check for conflicts: %w", err)
	}

	if len(conflicts) > 0 {
		log.Printf("[%s] Cannot auto-resolve: %d conflicts detected with %s", time.Now().Format(time.RFC3339), len(conflicts), remoteBranch)
		m.handleConflicts(conflicts)
		return fmt.Errorf("conflicts prevent automatic merge")
	}

	// Attempt the merge/pull
	if m.targetBranch != "" && m.targetBranch != currentBranch {
		// Cross-branch merge
		log.Printf("[%s] Auto-merging from remote branch '%s' into current branch '%s'", time.Now().Format(time.RFC3339), remoteBranch, currentBranch)
		if err := m.repo.MergeFromRemote(remoteBranch); err != nil {
			return fmt.Errorf("merge failed: %w", err)
		}
		log.Printf("[%s] Successfully merged from %s", time.Now().Format(time.RFC3339), remoteBranch)
		m.notifier.NotifyInSync(currentBranch)
	} else {
		// Same branch pull
		log.Printf("[%s] Auto-pulling changes into branch '%s'", time.Now().Format(time.RFC3339), currentBranch)
		if err := m.repo.Pull(); err != nil {
			return fmt.Errorf("pull failed: %w", err)
		}
		log.Printf("[%s] Successfully pulled changes", time.Now().Format(time.RFC3339))
		m.notifier.NotifyInSync(currentBranch)
	}

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
