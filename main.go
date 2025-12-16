package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const sharedSolutionsRepo = "https://raw.githubusercontent.com/CodingChallengesFYI/SharedSolutions/refs/heads/main/README.md"

type Input struct {
	Challenge string `json:"challenge" jsonschema:"the name of the Coding Challenge to look for"`
}

type Output struct {
	Solutions []string `json:"solutions,omitempty" jsonschema:"the solutions found"`
}

func fetchRaw(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/plain")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return "", fmt.Errorf("GET %s: status %s: %s", url, resp.Status, string(b))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func extractMarkdownLinkHrefByText(content, challenge string) ([]string, error) {
	pat := fmt.Sprintf(`(?i)\[\s*build your own %s.*?\s*]\(\s*([^) \t\r\n]+)\s*(?:[^)]*)\)`, regexp.QuoteMeta(challenge))
	re := regexp.MustCompile(pat)

	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	out := make([]string, 0, len(matches))

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		out = append(out, strings.TrimSpace(m[1]))
	}

	return out, nil
}

func search(ctx context.Context, _ *mcp.CallToolRequest, input Input) (
	*mcp.CallToolResult,
	Output,
	error,
) {
	if strings.TrimSpace(input.Challenge) == "" {
		return nil, Output{}, fmt.Errorf("Coding Challenge name cannot be empty")
	}
	solutions := make([]string, 0)

	content, err := fetchRaw(ctx, sharedSolutionsRepo)
	if err != nil {
		fmt.Println(err)
		return nil, Output{}, err
	}

	url, err := extractMarkdownLinkHrefByText(content, input.Challenge)
	if err != nil {
		return nil, Output{}, err
	}

	if len(url) > 0 {
		solutions = append(solutions, url...)
	}
	return nil, Output{Solutions: solutions}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "Coding Challenges Solutions", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "CodingChallengesSolutionFinder", Description: "Search the Coding Challenges Shared Solutions GitHub repo for shared solutions to a specific Coding Challenge"}, search)

	// Run the server over stdin/stdout, until the client disconnects.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
