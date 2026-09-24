package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"golang.org/x/net/html"
)

// AutoEnrichAsync runs the full "new grant" pipeline in the background:
// fetch the NOFO/source document from a URL, ingest it (chunks + embeddings),
// then extract structured requirements. Best-effort — failures are logged,
// the grant record itself stays valid.
func (s *GrantService) AutoEnrichAsync(grantID uuid.UUID, sourceURL string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.AutoEnrich(ctx, grantID, sourceURL); err != nil {
			log.Printf("[WARN] auto-enrich grant=%s: %v", grantID, err)
		} else {
			log.Printf("[INFO] auto-enrich complete grant=%s", grantID)
		}
	}()
}

// AutoEnrich fetches the grant's NOFO text from sourceURL (falling back to the
// stored application_url), ingests it, and extracts requirements.
func (s *GrantService) AutoEnrich(ctx context.Context, grantID uuid.UUID, sourceURL string) error {
	if sourceURL == "" {
		g, err := s.grants.GetByID(ctx, grantID)
		if err != nil {
			return fmt.Errorf("grant not found: %w", err)
		}
		if g.ApplicationURL != nil {
			sourceURL = *g.ApplicationURL
		}
	}
	if sourceURL == "" {
		return fmt.Errorf("no source URL to fetch NOFO from")
	}

	text, err := fetchNOFOText(ctx, sourceURL)
	if err != nil {
		return fmt.Errorf("fetch NOFO from %s: %w", sourceURL, err)
	}
	if len(strings.TrimSpace(text)) < 200 {
		return fmt.Errorf("fetched content too short (%d chars) — likely not a NOFO document", len(text))
	}

	if err := s.IngestNOFO(ctx, grantID, text); err != nil {
		return fmt.Errorf("ingest NOFO: %w", err)
	}
	if _, err := s.ExtractRequirements(ctx, grantID); err != nil {
		return fmt.Errorf("extract requirements: %w", err)
	}
	return nil
}

// IngestNOFOFile accepts an uploaded NOFO document (PDF or plain text),
// extracts its text, ingests it for RAG, and runs requirement extraction —
// the full "drop the document in" pipeline in one call.
func (s *GrantService) IngestNOFOFile(ctx context.Context, grantID uuid.UUID, filename string, data []byte) error {
	var text string
	if strings.HasSuffix(strings.ToLower(filename), ".pdf") || bytes.HasPrefix(data, []byte("%PDF")) {
		t, err := pdfToText(data)
		if err != nil {
			return fmt.Errorf("parse pdf: %w", err)
		}
		text = t
	} else {
		text = string(data)
	}
	if len(strings.TrimSpace(text)) < 200 {
		return fmt.Errorf("extracted text too short (%d chars) — upload the NOFO/solicitation document", len(text))
	}
	if err := s.IngestNOFO(ctx, grantID, text); err != nil {
		return fmt.Errorf("ingest NOFO: %w", err)
	}
	if _, err := s.ExtractRequirements(ctx, grantID); err != nil {
		return fmt.Errorf("extract requirements: %w", err)
	}
	return nil
}

// fetchNOFOText downloads a URL and returns plain text. PDFs are parsed;
// grants.gov detail pages resolve to their attachment PDFs; HTML pages either
// surface a linked NOFO PDF or fall back to stripped page text.
func fetchNOFOText(ctx context.Context, rawURL string) (string, error) {
	if id := grantsGovDetailID(rawURL); id != "" {
		if pdfURL, err := grantsGovPDF(ctx, id); err == nil && pdfURL != "" {
			rawURL = pdfURL
		} else {
			return "", fmt.Errorf("grants.gov opportunity %s: no downloadable NOFO attachment (%v)", id, err)
		}
	}

	body, contentType, err := fetchBytes(ctx, rawURL)
	if err != nil {
		return "", err
	}

	if strings.Contains(contentType, "pdf") || strings.HasSuffix(strings.ToLower(rawURL), ".pdf") {
		return pdfToText(body)
	}

	// HTML page: look for a linked NOFO/solicitation PDF first.
	doc := string(body)
	if pdfHref := findPDFLink(doc, rawURL); pdfHref != "" {
		pdfBody, _, err := fetchBytes(ctx, pdfHref)
		if err == nil {
			if t, err := pdfToText(pdfBody); err == nil && len(strings.TrimSpace(t)) > 200 {
				return t, nil
			}
		}
	}
	return htmlToText(doc), nil
}

func fetchBytes(ctx context.Context, rawURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ReadyGeneration/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 40<<20))
	return body, resp.Header.Get("Content-Type"), err
}

func pdfToText(body []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	plain, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract pdf text: %w", err)
	}
	b, err := io.ReadAll(plain)
	return string(b), err
}

var grantsGovDetailRe = regexp.MustCompile(`grants\.gov/search-results-detail/(\d+)`)

func grantsGovDetailID(rawURL string) string {
	m := grantsGovDetailRe.FindStringSubmatch(rawURL)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// grantsGovPDF resolves a grants.gov opportunity ID to its first PDF
// attachment via the public fetchOpportunity API.
func grantsGovPDF(ctx context.Context, oppID string) (string, error) {
	reqBody, _ := json.Marshal(map[string]string{"opportunityId": oppID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.grants.gov/v1/api/fetchOpportunity", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Folders []struct {
				Attachments []struct {
					ID       int64  `json:"id"`
					FileName string `json:"fileName"`
					MimeType string `json:"mimeType"`
				} `json:"synopsisAttachments"`
			} `json:"synopsisAttachmentFolders"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	for _, f := range result.Data.Folders {
		for _, a := range f.Attachments {
			if strings.HasSuffix(strings.ToLower(a.FileName), ".pdf") {
				return fmt.Sprintf("https://apply07.grants.gov/grantsws/rest/opportunity/document/download?attId=%d", a.ID), nil
			}
		}
	}
	return "", fmt.Errorf("no pdf attachment found")
}

// findPDFLink returns the first document link ending in .pdf that looks like a
// NOFO/solicitation, resolved against the page URL.
func findPDFLink(doc, pageURL string) string {
	base, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?:href|src)=["']([^"']+\.pdf[^"']*)["']`)
	best := ""
	for _, m := range re.FindAllStringSubmatch(doc, -1) {
		href := strings.TrimSpace(m[1])
		u, err := base.Parse(href)
		if err != nil {
			continue
		}
		abs := u.String()
		low := strings.ToLower(abs)
		if strings.Contains(low, "nofo") || strings.Contains(low, "solicitation") {
			return abs // best match — return immediately
		}
		if best == "" {
			best = abs
		}
	}
	return best
}

// htmlToText strips markup, scripts and styles, collapsing whitespace.
func htmlToText(doc string) string {
	z := html.NewTokenizer(strings.NewReader(doc))
	var b strings.Builder
	skip := 0
	for {
		tok := z.Next()
		switch tok {
		case html.ErrorToken:
			return whitespaceRe.ReplaceAllString(b.String(), " ")
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			if t.Data == "script" || t.Data == "style" || t.Data == "noscript" {
				skip++
			}
		case html.EndTagToken:
			t := z.Token()
			if (t.Data == "script" || t.Data == "style" || t.Data == "noscript") && skip > 0 {
				skip--
			}
		case html.TextToken:
			if skip == 0 {
				b.WriteString(z.Token().Data)
				b.WriteByte(' ')
			}
		}
	}
}

var whitespaceRe = regexp.MustCompile(`\s+`)
