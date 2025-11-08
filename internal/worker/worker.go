package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/your-username/queuectl/internal/config"
	"github.com/your-username/queuectl/internal/store"
)

// Worker processes jobs from the queue.
type Worker struct {
	ID    int
	Store store.Store
	Cfg   *config.Config
}

func NewWorker(id int, s store.Store, cfg *config.Config) *Worker {
	return &Worker{
		ID:    id,
		Store: s,
		Cfg:   cfg,
	}
}

// Run starts the worker's processing loop.
func (w *Worker) Run(ctx context.Context) {
	log.Printf("Worker %d started", w.ID)
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", w.ID)
			return
		default:
			job, err := w.Store.FindAndLockJob()
			if err != nil {
				log.Printf("Worker %d: Error finding job: %v", w.ID, err)
				time.Sleep(1 * time.Second) // Avoid busy-looping on DB error
				continue
			}

			if job == nil {
				time.Sleep(1 * time.Second) // No job found, wait a bit
				continue
			}

			w.processJob(job)
		}
	}
}

func (w *Worker) processJob(job *store.Job) {
	log.Printf("Worker %d: Processing job %s (Attempt %d)", w.ID, job.ID, job.Attempts)

	// The command can be complex, so we use "sh -c" to execute it
	cmd := exec.Command("sh", "-c", job.Command)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Worker %d: Job %s failed: %v. Output: %s", w.ID, job.ID, err, string(output))
		w.handleFailure(job)
	} else {
		log.Printf("Worker %d: Job %s completed successfully. Output: %s", w.ID, job.ID, string(output))
		job.State = store.StateCompleted
		if err := w.Store.UpdateJob(job); err != nil {
			log.Printf("Worker %d: Error updating completed job %s: %v", w.ID, job.ID, err)
		}
	}
}

func (w *Worker) handleFailure(job *store.Job) {
	if job.Attempts >= job.MaxRetries {
		log.Printf("Worker %d: Job %s has reached max retries. Moving to DLQ.", w.ID, job.ID)
		job.State = store.StateDead
	} else {
		job.State = store.StateFailed // Intermediate state, will be set to pending
		backoffDuration := time.Duration(math.Pow(w.Cfg.BackoffBase, float64(job.Attempts))) * time.Second
		job.NextRunAt = time.Now().UTC().Add(backoffDuration)
		job.State = store.StatePending // Set back to pending for the next run
		log.Printf("Worker %d: Job %s will be retried in %v.", w.ID, job.ID, backoffDuration)
	}

	if err := w.Store.UpdateJob(job); err != nil {
		log.Printf("Worker %d: Error updating failed job %s: %v", w.ID, job.ID, err)
	}
}

// Manager orchestrates multiple workers.
type Manager struct {
	Count int
	Store store.Store
	Cfg   *config.Config
}

func NewManager(count int, s store.Store, cfg *config.Config) *Manager {
	return &Manager{
		Count: count,
		Store: s,
		Cfg:   cfg,
	}
}

func (m *Manager) Start() {
	pidFile, err := getPidFilePath()
	if err != nil {
		log.Fatalf("Error getting PID file path: %v", err)
	}
	if _, err := os.Stat(pidFile); err == nil {
		log.Fatalf("Workers already running or PID file stale. Please run 'queuectl worker stop' or remove %s", pidFile)
	}

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer os.Remove(pidFile)

	log.Printf("Starting %d workers...", m.Count)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for i := 1; i <= m.Count; i++ {
		wg.Add(1)
		worker := NewWorker(i, m.Store, m.Cfg)
		go func() {
			defer wg.Done()
			worker.Run(ctx)
		}()
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received, stopping workers gracefully...")
	cancel()
	wg.Wait()
	log.Println("All workers have stopped.")
}

func StopWorkers() error {
	pidFile, err := getPidFilePath()
	if err != nil {
		return fmt.Errorf("error getting PID file path: %w", err)
	}

	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("workers not running or PID file not found: %w", err)
	}

	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		return fmt.Errorf("invalid PID in PID file: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		// On Unix, FindProcess always succeeds, so we check the error from Signal
		// but this could be an issue on Windows.
		return fmt.Errorf("could not find process with PID %d: %w", pid, err)
	}

	log.Printf("Sending SIGTERM to worker process with PID %d", pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// If the process doesn't exist, we might get an error.
		// We can consider this a success and clean up the PID file.
		if os.IsNotExist(err) {
			log.Println("Worker process not found. Removing stale PID file.")
			os.Remove(pidFile)
			return nil
		}
		return fmt.Errorf("failed to send signal to process %d: %w", pid, err)
	}

	// The PID file is removed by the daemon process itself upon clean exit.
	// We can add a timeout here to force-remove it if needed.
	log.Println("Stop signal sent. Workers should shut down shortly.")
	return nil
}

func getPidFilePath() (string, error) {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "queuectl.pid"), nil
}

func GetActiveWorkerCount() int {
	pidFile, err := getPidFilePath()
	if err != nil {
		return 0
	}
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return 0
	}
	// A more robust check would be to see if the process is actually running
	// but for a summary, checking the PID file existence is a good start.
	return 1 // We know a manager process is running, but not the worker count inside it.
}
