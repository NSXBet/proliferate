package pullrequest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/nsxbet/proliferate/pkg/mygit"
	"github.com/nsxbet/proliferate/pkg/types"
	"gopkg.in/yaml.v3"
)

type PRStatusManager struct {
	statusDir string
	printer   types.Printer
	mu        sync.Mutex
	status    NamespacedStatus
}

func NewPRStatusManager(statusDir string, printer types.Printer) *PRStatusManager {
	return &PRStatusManager{
		statusDir: statusDir,
		printer:   printer,
		status:    make(NamespacedStatus),
	}
}

type PRStatus = types.PRStatus

type NamespacedStatus map[string]map[string]PRStatus

func (m *PRStatusManager) SaveStatus(namespace string, status PRStatus) error {
	allStatus, err := m.loadAll()
	if err != nil {
		return err
	}

	if allStatus[namespace] == nil {
		allStatus[namespace] = make(map[string]PRStatus)
	}

	allStatus[namespace][status.Name] = status
	return m.saveAll(allStatus)
}

func (m *PRStatusManager) GetNamespaces() ([]string, error) {
	status, err := m.loadAll()
	if err != nil {
		return nil, err
	}

	var namespaces []string
	for ns := range status {
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}

func (m *PRStatusManager) GetByNamespace(namespace string) (map[string]PRStatus, error) {
	status, err := m.loadAll()
	if err != nil {
		return nil, err
	}

	return status[namespace], nil
}

func (m *PRStatusManager) loadAll() (NamespacedStatus, error) {
	statusFile := filepath.Join(m.statusDir, "status.yaml")
	status := make(NamespacedStatus)

	if _, err := os.Stat(statusFile); os.IsNotExist(err) {
		return status, nil
	}

	data, err := os.ReadFile(statusFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read status file: %v", err)
	}

	if err := yaml.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status file: %v", err)
	}

	return status, nil
}

func (m *PRStatusManager) saveAll(status NamespacedStatus) error {
	if err := os.MkdirAll(m.statusDir, 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %v", err)
	}

	data, err := yaml.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %v", err)
	}

	if err := os.WriteFile(filepath.Join(m.statusDir, "status.yaml"), data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %v", err)
	}

	return nil
}

func (m *PRStatusManager) DisplayNamespacesSummary() error {
	namespaces, err := m.GetNamespaces()
	if err != nil {
		return err
	}

	counts := make(map[string]int)
	for _, namespace := range namespaces {
		prs, err := m.GetByNamespace(namespace)
		if err != nil {
			return err
		}
		counts[namespace] = len(prs)
	}

	m.printer.PrintNamespacesSummary(namespaces, counts)
	return nil
}

func (m *PRStatusManager) DisplayNamespaceDetails(ctx context.Context, namespace string, git *mygit.Git) error {
	prs, err := m.GetByNamespace(namespace)
	if err != nil {
		return err
	}
	if prs == nil {
		return fmt.Errorf("namespace %s not found", namespace)
	}

	m.printer.PrintNamespaceHeader(namespace)

	type prResult struct {
		name  string
		pr    PRStatus
		state string
		err   error
	}

	resultChan := make(chan prResult, len(prs))
	var wg sync.WaitGroup

	for name, pr := range prs {
		wg.Add(1)
		go func(name string, pr PRStatus) {
			defer wg.Done()
			owner, repoName, err := git.ParseRepoString(pr.Repository)
			if err != nil {
				resultChan <- prResult{name: name, err: err}
				return
			}

			state, err := git.GetPRStatus(ctx, owner, repoName, pr.PRNumber)
			resultChan <- prResult{name: name, pr: pr, state: state, err: err}
		}(name, pr)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results in order received
	for result := range resultChan {
		if result.err != nil {
			m.printer.PrintError("\n PR: %s (Failed: %v)\n", result.name, result.err)
			continue
		}
		m.printer.PrintPRStatus(result.name, result.pr, result.state)
	}

	return nil
}

func (m *PRStatusManager) UpdatePRStatus(namespace, name string, updateFn func(*types.PRStatus)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, err := m.loadAll()
	if err != nil {
		return err
	}

	if status[namespace] == nil {
		status[namespace] = make(map[string]PRStatus)
	}

	prStatus := status[namespace][name]
	updateFn(&prStatus)
	status[namespace][name] = prStatus

	return m.saveAll(status)
}
