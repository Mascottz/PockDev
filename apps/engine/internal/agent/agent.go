// Package agent is the Pocket AI Agent: it wakes up only when an incident is
// open, reads the crash and the code, writes a plain English diagnosis, and
// alerts the developer over WhatsApp or email.
package agent

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/notify"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
	"github.com/Mascottz/PockDev/apps/engine/internal/worker"
)

// Agent runs incident diagnosis and the phone driven fix loop.
type Agent struct {
	Store   store.Store
	LLM     *LLM
	GitHub  *GitHub
	Notify  *notify.Client
	MuteFor time.Duration
	Now     func() time.Time
	Logf    func(format string, args ...any)

	queue   chan string
	started sync.Once
}

// Run processes the incident queue until ctx is cancelled.
func (a *Agent) Run(ctx context.Context) {
	a.started.Do(func() {
		a.queue = make(chan string, 32)
	})
	if a.Logf == nil {
		a.Logf = log.Printf
	}
	if a.MuteFor <= 0 {
		a.MuteFor = time.Hour
	}
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-a.queue:
			if err := a.Diagnose(ctx, id); err != nil {
				a.Logf("agent: diagnose %s: %v", id, err)
			}
		}
	}
}

// Trigger queues an incident for diagnosis. Safe to call before Run.
func (a *Agent) Trigger(incidentID string) {
	a.started.Do(func() {
		a.queue = make(chan string, 32)
	})
	select {
	case a.queue <- incidentID:
	default:
		a.Logf("agent: queue full, incident %s skipped", incidentID)
	}
}

// Diagnose builds the plain English summary and alerts the developer.
func (a *Agent) Diagnose(ctx context.Context, incidentID string) error {
	inc, err := a.Store.GetIncident(ctx, incidentID)
	if err != nil {
		return err
	}
	project, err := a.Store.GetProject(ctx, inc.ProjectID)
	if err != nil {
		return err
	}
	paused, err := a.Store.ListDeliveries(ctx, model.DeliveryFilter{IncidentID: inc.ID, Status: model.DeliveryPaused})
	if err == nil {
		inc.PausedCount = len(paused)
	}

	summary := a.fallbackSummary(inc, project)
	if a.LLM != nil && a.LLM.BaseURL != "" {
		if text, err := a.LLM.Complete(ctx, diagnosisSystem, a.diagnosisUser(ctx, inc, project)); err == nil {
			text = strings.TrimSpace(text)
			if text != "" {
				summary = text
			}
		} else {
			a.Logf("agent: llm failed, using fallback summary: %v", err)
		}
	}

	inc.PlainSummary = summary
	inc.LastSeenAt = time.Now()
	if a.Now != nil {
		inc.LastSeenAt = a.Now()
	}
	menu := fmt.Sprintf(" %d webhooks are safely paused. Reply 1 to open a fix PR, 2 to replay paused webhooks, or 3 to mute for 1 hour.", inc.PausedCount)
	if inc.PausedCount == 1 {
		menu = " 1 webhook is safely paused. Reply 1 to open a fix PR, 2 to replay paused webhooks, or 3 to mute for 1 hour."
	}
	if inc.PausedCount == 0 {
		menu = " Reply 1 to open a fix PR, 2 to replay paused webhooks, or 3 to mute for 1 hour."
	}
	tag := inc.ID
	if len(tag) > 4 {
		tag = tag[len(tag)-4:]
	}
	message := "[#" + tag + "] " + summary + menu

	if a.Notify != nil {
		if project.WhatsAppPhone != "" && a.Notify.WhatsAppConfigured() {
			if err := a.Notify.SendWhatsApp(ctx, project.WhatsAppPhone, message); err != nil {
				a.Logf("agent: whatsapp alert failed (to=%s): %v", project.WhatsAppPhone, err)
			} else {
				a.Logf("agent: whatsapp alert sent to %s for %s", project.WhatsAppPhone, inc.ID)
				now := time.Now()
				inc.LastAlertPhone = "+" + strings.TrimLeft(strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(project.WhatsAppPhone)), "+")
				inc.LastAlertAt = &now
			}
		} else {
			a.Logf("agent: whatsapp skipped (project_phone=%q api_configured=%v)", project.WhatsAppPhone, a.Notify.WhatsAppConfigured())
		}
		if a.Notify.EmailConfigured() {
			subject := "PockDev alert: " + inc.Title
			if err := a.Notify.SendEmail(ctx, subject, message); err != nil {
				a.Logf("agent: email alert failed: %v", err)
			}
		}
	}
	return a.Store.UpdateIncident(ctx, inc)
}

const diagnosisSystem = "You are the PockDev pocket agent, a calm senior engineer texting a busy developer. " +
	"Write one short paragraph in plain English with no jargon. " +
	"Say which service crashed, where the crash is (endpoint, file, line), what the code tried to do, and what it means. " +
	"Never use em dashes or en dashes. Do not mention that you are an AI or a model. Keep it under 70 words."

func (a *Agent) diagnosisUser(ctx context.Context, inc *model.Incident, project *model.Project) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Service: %s\nProblem: %s\nFile: %s\n", project.Name, inc.Title, inc.FilePath)
	if inc.Origin == model.OriginWebhookDeadLetter {
		b.WriteString("Trigger: webhooks stopped reaching the target and are now paused.\n")
	}
	if ev := a.latestEvent(ctx, inc); ev != nil {
		fmt.Fprintf(&b, "Crash message: %s\nLine: %d\nStack:\n%s\n", ev.Message, ev.Line, clip(ev.Stack, 1500))
	}
	if project.GitHubRepo != "" && a.GitHub != nil && a.GitHub.Token != "" {
		if content, err := a.GitHub.FileContent(ctx, project.GitHubRepo, inc.FilePath); err == nil {
			fmt.Fprintf(&b, "Current file content:\n%s\n", clip(content, 6000))
		}
		if commits, err := a.GitHub.RecentCommits(ctx, project.GitHubRepo, 5); err == nil && len(commits) > 0 {
			b.WriteString("Recent commits:\n")
			for _, c := range commits {
				fmt.Fprintf(&b, "- %s %s\n", c.SHA[:7], c.Message)
			}
		}
	}
	return b.String()
}

func (a *Agent) latestEvent(ctx context.Context, inc *model.Incident) *model.ErrorEvent {
	if inc.Origin == model.OriginErrorSpike && inc.OriginRef != "" {
		if ev, err := a.Store.LatestErrorEvent(ctx, inc.OriginRef); err == nil {
			return ev
		}
	}
	if inc.Origin == model.OriginWebhookDeadLetter && inc.OriginRef != "" {
		if ev, err := a.Store.ErrorEventByDelivery(ctx, inc.OriginRef); err == nil {
			return ev
		}
	}
	return nil
}

func (a *Agent) fallbackSummary(inc *model.Incident, project *model.Project) string {
	if inc.Origin == model.OriginWebhookDeadLetter {
		return fmt.Sprintf("Webhooks for %s are not reaching the target, so they are paused safely instead of lost.", project.Name)
	}
	where := inc.FilePath
	if where == "" {
		where = "the app"
	}
	return fmt.Sprintf("Crash in %s at %s. %s", project.Name, where, inc.Title)
}

// DraftPR patches the crashing file with the LLM and opens a fix PR.
func (a *Agent) DraftPR(ctx context.Context, incidentID string) (*model.Incident, error) {
	inc, err := a.Store.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	project, err := a.Store.GetProject(ctx, inc.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.GitHubRepo == "" {
		return nil, fmt.Errorf("project has no GitHub repo set")
	}
	if a.GitHub == nil || a.GitHub.Token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN not set")
	}
	if a.LLM == nil || a.LLM.BaseURL == "" {
		return nil, fmt.Errorf("LLM_BASE_URL not set")
	}
	if inc.FilePath == "" {
		return nil, fmt.Errorf("incident has no file path to fix")
	}

	inc.Status = model.IncidentFixing
	inc.LastSeenAt = time.Now()
	if err := a.Store.UpdateIncident(ctx, inc); err != nil {
		return nil, err
	}

	content, err := a.GitHub.FileContent(ctx, project.GitHubRepo, inc.FilePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", inc.FilePath, err)
	}
	ev := a.latestEvent(ctx, inc)
	var crash strings.Builder
	fmt.Fprintf(&crash, "Service: %s\nProblem: %s\nFile: %s\n", project.Name, inc.Title, inc.FilePath)
	if ev != nil {
		fmt.Fprintf(&crash, "Crash message: %s\nLine: %d\nStack:\n%s\n", ev.Message, ev.Line, clip(ev.Stack, 1500))
	}
	fmt.Fprintf(&crash, "Current file content:\n%s\n", content)

	reply, err := a.LLM.Complete(ctx, fixSystem, crash.String())
	if err != nil {
		return nil, fmt.Errorf("llm fix failed: %w", err)
	}
	fixed := extractCodeBlock(reply)
	if fixed == "" {
		return nil, fmt.Errorf("llm did not return a fixed file")
	}

	branch := "pockdev/fix-" + strings.ToLower(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(inc.ID, "-"))
	if _, err := a.GitHub.CreateBranch(ctx, project.GitHubRepo, branch); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}
	commitMsg := "Mascott, fix " + clip(inc.Title, 80)
	if err := a.GitHub.CommitFile(ctx, project.GitHubRepo, branch, inc.FilePath, fixed, commitMsg); err != nil {
		return nil, fmt.Errorf("commit fix: %w", err)
	}
	prTitle := "Mascott, fix " + clip(inc.Title, 80)
	prBody := "Suggested fix for: " + inc.Title + "\n\nFile: " + inc.FilePath
	prURL, err := a.GitHub.OpenPR(ctx, project.GitHubRepo, prTitle, branch, prBody)
	if err != nil {
		return nil, fmt.Errorf("open PR: %w", err)
	}

	inc.PRURL = prURL
	inc.LastSeenAt = time.Now()
	if err := a.Store.UpdateIncident(ctx, inc); err != nil {
		return nil, err
	}
	return inc, nil
}

const fixSystem = "You are the PockDev pocket agent fixing a crash for a busy developer. " +
	"Reply with ONLY the complete corrected file content inside one fenced code block. " +
	"Change as little as possible while fixing the crash. " +
	"Never use em dashes or en dashes in the code or comments. Do not explain anything outside the code block."

var codeBlock = regexp.MustCompile("(?s)```[a-zA-Z0-9]*\\n(.*?)```")

func extractCodeBlock(s string) string {
	m := codeBlock.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return strings.TrimRight(m[1], "\n") + "\n"
}

// ReplayAll replays every paused or failed delivery tied to the incident.
func (a *Agent) ReplayAll(ctx context.Context, incidentID string) (*model.Incident, int, error) {
	inc, err := a.Store.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, 0, err
	}
	n, err := worker.ReplayAll(ctx, a.Store, incidentID)
	if err != nil {
		return nil, 0, err
	}
	inc.PausedCount = 0
	inc.LastSeenAt = time.Now()
	if err := a.Store.UpdateIncident(ctx, inc); err != nil {
		return nil, n, err
	}
	return inc, n, nil
}

// Resolve marks the incident as handled.
func (a *Agent) Resolve(ctx context.Context, incidentID string) (*model.Incident, error) {
	inc, err := a.Store.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	inc.Status = model.IncidentResolved
	inc.LastSeenAt = time.Now()
	if err := a.Store.UpdateIncident(ctx, inc); err != nil {
		return nil, err
	}
	return inc, nil
}

// Mute quiets the incident for the mute window.
func (a *Agent) Mute(ctx context.Context, incidentID string) (*model.Incident, error) {
	inc, err := a.Store.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	dur := a.MuteFor
	if dur <= 0 {
		dur = time.Hour
	}
	until := time.Now().Add(dur)
	inc.Status = model.IncidentMuted
	inc.MuteUntil = &until
	inc.LastSeenAt = time.Now()
	if err := a.Store.UpdateIncident(ctx, inc); err != nil {
		return nil, err
	}
	return inc, nil
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}
