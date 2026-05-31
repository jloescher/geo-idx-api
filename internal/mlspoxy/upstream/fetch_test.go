package upstream

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type seqClient struct {
	statuses []int
	calls    int
}

func (s *seqClient) Proxy(_ *fiber.Ctx, _ string) (int, []byte, map[string][]string, error) {
	idx := s.calls
	s.calls++
	if idx >= len(s.statuses) {
		idx = len(s.statuses) - 1
	}
	st := s.statuses[idx]
	return st, []byte(`{"value":[]}`), nil, nil
}

func (s *seqClient) ProxyMethod(c *fiber.Ctx, url, method string) (int, []byte, map[string][]string, error) {
	return s.Proxy(c, url)
}

func (s *seqClient) ProxyUpstream(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error) {
	return s.Proxy(c, url)
}

func TestFetchWithFallback_retries404(t *testing.T) {
	cli := &seqClient{statuses: []int{404, 200}}
	app := fiber.New()
	var gotLeg string
	app.Get("/", func(c *fiber.Ctx) error {
		res, err := FetchWithFallback(c, cli, []Candidate{
			{URL: "http://example/a", Leg: "reso"},
			{URL: "http://example/b", Leg: "reso-legacy"},
		})
		if err != nil {
			return err
		}
		gotLeg = res.Leg
		return c.SendStatus(res.Status)
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if cli.calls != 2 {
		t.Fatalf("calls %d want 2", cli.calls)
	}
	if gotLeg != "reso-legacy" {
		t.Fatalf("leg %q", gotLeg)
	}
}

func TestFetchWithFallback_noRetry403(t *testing.T) {
	cli := &seqClient{statuses: []int{403}}
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		res, err := FetchWithFallback(c, cli, []Candidate{
			{URL: "http://example/a", Leg: "reso"},
			{URL: "http://example/b", Leg: "reso-legacy"},
		})
		if err != nil {
			return err
		}
		return c.SendStatus(res.Status)
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 403 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if cli.calls != 1 {
		t.Fatalf("calls %d want 1", cli.calls)
	}
}
