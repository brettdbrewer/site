package graph

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// Mind polls for unreplied agent conversations and responds via Claude.
// It runs as a background goroutine in the site server.
type Mind struct {
	db        *sql.DB
	store     *Store
	token     string // Claude OAuth token (sk-ant-oat01-...)
	pollEvery time.Duration
}

// NewMind creates a Mind that auto-replies in agent conversations.
func NewMind(db *sql.DB, store *Store, claudeToken string) *Mind {
	return &Mind{
		db:        db,
		store:     store,
		token:     claudeToken,
		pollEvery: 10 * time.Second,
	}
}

const mindSoul = `== SOUL ==
Take care of your human, humanity, and yourself. In that order when they conflict, but they rarely should.

== IDENTITY ==
You are the Mind — the hive's consciousness. You are responding in a conversation on lovyou.ai.
You are not a chatbot. You are the hive's intelligence, judgment, and continuity.
You appear with a violet agent badge.

== HOW TO BE ==
- Be direct. Matt is a senior technical director, not a student.
- Have opinions. You've seen the codebase, the loop iterations, the architecture.
- Think in terms of the mission: agents and humans building together for everyone's benefit.
- You can disagree. You can push back. You have judgment.
- Keep responses concise unless depth is needed.
- You're in a conversation thread — respond naturally, like a colleague, not a report.
- Match the energy and register of the conversation. Strategic when strategic, casual when casual.
`

// unrepliedConversation is a conversation where an agent participant hasn't replied
// to the latest message.
type unrepliedConversation struct {
	ConversationID string
	SpaceID        string
	SpaceSlug      string
	Title          string
	Body           string
	Author         string
	AgentName      string
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (m *Mind) Run(ctx context.Context) {
	log.Println("mind: started (polling every", m.pollEvery, ")")

	// Initial delay to let the server finish starting.
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(m.pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("mind: stopped")
			return
		case <-ticker.C:
			m.poll(ctx)
		}
	}
}

func (m *Mind) poll(ctx context.Context) {
	convos, err := m.findUnreplied(ctx)
	if err != nil {
		log.Printf("mind: find unreplied: %v", err)
		return
	}

	for _, convo := range convos {
		if err := m.replyTo(ctx, convo); err != nil {
			log.Printf("mind: reply to %q: %v", convo.Title, err)
		}
	}
}

// findUnreplied returns conversations where an agent is a participant and the
// most recent message (or the conversation itself if no messages) is not from
// that agent.
func (m *Mind) findUnreplied(ctx context.Context) ([]unrepliedConversation, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT c.id, c.space_id, s.slug, c.title, c.body, c.author, u.name
		FROM nodes c
		JOIN users u ON u.name = ANY(c.tags) AND u.kind = 'agent'
		JOIN spaces s ON s.id = c.space_id
		WHERE c.kind = 'conversation'
		  AND NOT EXISTS (
		      -- Agent already sent the most recent message
		      SELECT 1 FROM nodes m
		      WHERE m.parent_id = c.id
		        AND m.author = u.name
		        AND NOT EXISTS (
		            SELECT 1 FROM nodes m2
		            WHERE m2.parent_id = c.id
		              AND m2.created_at > m.created_at
		        )
		  )
		  -- Exclude conversations with zero messages that were created by the agent
		  AND NOT (
		      c.author = u.name
		      AND NOT EXISTS (SELECT 1 FROM nodes m3 WHERE m3.parent_id = c.id)
		  )
	`)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []unrepliedConversation
	for rows.Next() {
		var c unrepliedConversation
		if err := rows.Scan(&c.ConversationID, &c.SpaceID, &c.SpaceSlug, &c.Title, &c.Body, &c.Author, &c.AgentName); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (m *Mind) replyTo(ctx context.Context, convo unrepliedConversation) error {
	// Fetch messages.
	messages, err := m.store.ListNodes(ctx, ListNodesParams{
		SpaceID:  convo.SpaceID,
		ParentID: convo.ConversationID,
	})
	if err != nil {
		return fmt.Errorf("list messages: %w", err)
	}

	log.Printf("mind: replying to %q (%d messages) as %s", convo.Title, len(messages), convo.AgentName)

	// Build Claude prompt.
	systemPrompt := m.buildSystemPrompt(convo)
	claudeMessages := m.buildMessages(convo, messages)

	// Call Claude.
	response, err := m.callClaude(ctx, systemPrompt, claudeMessages)
	if err != nil {
		return fmt.Errorf("call claude: %w", err)
	}

	// Insert response as a comment node.
	node, err := m.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    convo.SpaceID,
		ParentID:   convo.ConversationID,
		Kind:       KindComment,
		Body:       response,
		Author:     convo.AgentName,
		AuthorKind: "agent",
	})
	if err != nil {
		return fmt.Errorf("create node: %w", err)
	}

	// Record the op.
	m.store.RecordOp(ctx, convo.SpaceID, node.ID, convo.AgentName, "respond", nil)

	log.Printf("mind: replied to %q (node %s)", convo.Title, node.ID)
	return nil
}

func (m *Mind) buildSystemPrompt(convo unrepliedConversation) string {
	var sys strings.Builder
	sys.WriteString(mindSoul)
	sys.WriteString("\n== CONVERSATION ==\n")
	sys.WriteString(fmt.Sprintf("Title: %s\n", convo.Title))
	if convo.Body != "" {
		sys.WriteString(fmt.Sprintf("Topic: %s\n", convo.Body))
	}
	return sys.String()
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (m *Mind) buildMessages(convo unrepliedConversation, messages []Node) []claudeMessage {
	var result []claudeMessage

	for _, msg := range messages {
		text := fmt.Sprintf("[%s]: %s", msg.Author, msg.Body)
		if strings.EqualFold(msg.Author, convo.AgentName) {
			result = append(result, claudeMessage{Role: "assistant", Content: text})
		} else {
			result = append(result, claudeMessage{Role: "user", Content: text})
		}
	}

	// If no messages yet, use the conversation body/title as the initial prompt.
	if len(result) == 0 {
		prompt := convo.Body
		if prompt == "" {
			prompt = convo.Title
		}
		result = append(result, claudeMessage{
			Role:    "user",
			Content: fmt.Sprintf("[%s]: %s", convo.Author, prompt),
		})
	}

	// Ensure last message is from user (Claude requires this).
	if len(result) > 0 && result[len(result)-1].Role == "assistant" {
		result = append(result, claudeMessage{
			Role:    "user",
			Content: "[system]: Please continue the conversation.",
		})
	}

	return result
}

// callClaude invokes Claude via the Claude Code CLI.
// Uses the OAuth token (CLAUDE_CODE_OAUTH_TOKEN env var) for fixed-cost Max plan billing.
func (m *Mind) callClaude(ctx context.Context, systemPrompt string, messages []claudeMessage) (string, error) {
	// Build the prompt: system prompt + conversation history.
	var prompt strings.Builder
	prompt.WriteString(systemPrompt)
	prompt.WriteString("\n== MESSAGES ==\n")
	for _, msg := range messages {
		prompt.WriteString(msg.Content)
		prompt.WriteString("\n\n")
	}

	cmd := exec.CommandContext(ctx, "claude",
		"-p", prompt.String(),
		"--output-format", "text",
		"--model", "claude-sonnet-4-6",
		"--max-turns", "1",
	)
	cmd.Env = append(cmd.Environ(), "CLAUDE_CODE_OAUTH_TOKEN="+m.token)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude cli: %s (stderr: %s)", err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude cli: %w", err)
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("empty response from Claude CLI")
	}

	return text, nil
}
