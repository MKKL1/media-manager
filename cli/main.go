package main

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "mm",
		Usage: "media manager client",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "server", Value: "http://localhost:3000", Sources: cli.EnvVars("MM_SERVER")},
			&cli.StringFlag{Name: "token", Usage: "bearer token", Sources: cli.EnvVars("MM_TOKEN")},
		},
		Commands: []*cli.Command{
			{
				Name:  "whoami",
				Usage: "who the token signs in as",
				Action: func(ctx context.Context, c *cli.Command) error {
					return show(call(ctx, c, "GET", "/api/v1/whoami", nil))
				},
			},
			{
				Name:      "can-i",
				Usage:     "whether you may do something",
				ArgsUsage: "<verb> <resource> [name]",
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.NArg() < 2 {
						return cli.ShowSubcommandHelp(c)
					}
					return show(call(ctx, c, "POST", "/api/v1/accessreviews", map[string]string{
						"verb": c.Args().Get(0), "resource": c.Args().Get(1), "name": c.Args().Get(2),
					}))
				},
			},
			{
				Name:  "providers",
				Usage: "metadata providers",
				Flags: []cli.Flag{&cli.StringFlag{Name: "tag", Usage: "only providers that tag with this (default:anime)"}},
				Action: func(ctx context.Context, c *cli.Command) error {
					path := "/api/v1/providers"
					if tag := c.String("tag"); tag != "" {
						path += "?tag=" + url.QueryEscape(tag)
					}
					return show(call(ctx, c, "GET", path, nil))
				},
			},
			{
				Name:      "search",
				Usage:     "what a provider finds",
				ArgsUsage: "<provider> <query...>",
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.NArg() < 2 {
						return cli.ShowSubcommandHelp(c)
					}
					query := strings.Join(c.Args().Tail(), " ")
					return show(call(ctx, c, "GET", "/api/v1/providers/"+url.PathEscape(c.Args().First())+"/search?q="+url.QueryEscape(query), nil))
				},
			},
			{
				Name:      "fetch",
				Usage:     "fetch a work into the library",
				ArgsUsage: "<provider> <ref>",
				Flags:     []cli.Flag{&cli.StringFlag{Name: "arrangement", Usage: "the provider's default if empty"}},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.NArg() != 2 {
						return cli.ShowSubcommandHelp(c)
					}
					return show(call(ctx, c, "POST", "/api/v1/providers/"+url.PathEscape(c.Args().First())+"/fetch", map[string]string{
						"ref": c.Args().Get(1), "arrangement": c.String("arrangement"),
					}))
				},
			},
			{
				Name:      "entry",
				Usage:     "one entry of the library",
				ArgsUsage: "<id>",
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.NArg() != 1 {
						return cli.ShowSubcommandHelp(c)
					}
					return show(call(ctx, c, "GET", "/api/v1/entries/"+url.PathEscape(c.Args().First()), nil))
				},
			},
			{
				Name:      "view",
				Usage:     "a work as you choose to view it, named by its metadata type",
				ArgsUsage: "<work id>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "arrangement", Usage: `"dvd", or "tvdb/default" for another provider's; the one it was fetched in if empty`},
					&cli.StringFlag{Name: "type", Usage: "metadata type (default:anime); the first of the work's tags that has one if empty"},
					&cli.BoolFlag{Name: "json", Usage: "print the view with its values as JSON"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.NArg() != 1 {
						return cli.ShowSubcommandHelp(c)
					}
					work := c.Args().First()
					q := url.Values{}
					if a := c.String("arrangement"); a != "" {
						q.Set("arrangement", a)
					}
					if t := c.String("type"); t != "" {
						q.Set("type", t)
					}
					body, err := call(ctx, c, "GET", "/api/v1/entries/"+url.PathEscape(work)+"/view?"+q.Encode(), nil)
					var nf notFound
					if errors.As(err, &nf) && nf.Kind == "arrangement" {
						provider, ref, _ := strings.Cut(work, ":")
						name := nf.Name[strings.Index(nf.Name, "/")+1:]
						return fmt.Errorf("%w\nfetch it first: mm fetch %s %s --arrangement %s", err, provider, ref, name)
					}
					if err != nil || c.Bool("json") {
						return show(body, err)
					}
					var answer struct{ Data viewed }
					if err := json.Unmarshal(body, &answer); err != nil {
						return err
					}
					fmt.Printf("# %s as %s\n", answer.Data.Arrangement, cmp.Or(answer.Data.Type, "no metadata type"))
					answer.Data.Work.print("")
					return nil
				},
			},
			{
				Name:  "plugins",
				Usage: "plugins named in the server's config",
				Action: func(ctx context.Context, c *cli.Command) error {
					return show(call(ctx, c, "GET", "/api/v1/plugins", nil))
				},
				Commands: []*cli.Command{{
					Name:      "reload",
					Usage:     "load a plugin's file again, or every plugin's; the old one stays on an error",
					ArgsUsage: "[name]",
					Action: func(ctx context.Context, c *cli.Command) error {
						names := c.Args().Slice()
						if len(names) == 0 {
							body, err := call(ctx, c, "GET", "/api/v1/plugins", nil)
							if err != nil {
								return err
							}
							var answer struct{ Data []string }
							if err := json.Unmarshal(body, &answer); err != nil {
								return err
							}
							names = answer.Data
						}
						for _, name := range names {
							if _, err := call(ctx, c, "POST", "/api/v1/plugins/"+url.PathEscape(name)+"/reload", nil); err != nil {
								return fmt.Errorf("%s: %w", name, err)
							}
							fmt.Println("reloaded", name)
						}
						return nil
					},
				}},
			},
			{
				Name:  "watch",
				Usage: "print changes as they happen, one JSON event per line",
				Action: func(ctx context.Context, c *cli.Command) error {
					res, err := send(ctx, c, "GET", "/api/v1/events?watch=true", nil)
					if err != nil {
						return err
					}
					defer res.Body.Close()
					lines := bufio.NewScanner(res.Body)
					for lines.Scan() {
						if data, ok := strings.CutPrefix(lines.Text(), "data: "); ok {
							fmt.Println(data)
						}
					}
					return lines.Err()
				},
			},
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "mm:", err)
		os.Exit(1)
	}
}

type viewed struct {
	Arrangement, Type string
	Work              viewedEntry
}

type viewedEntry struct {
	ID, Name string
	Entries  []viewedEntry
}

func (e viewedEntry) print(indent string) {
	fmt.Println(indent + cmp.Or(e.Name, e.ID))
	for _, child := range e.Entries {
		child.print(indent + "  ")
	}
}

type notFound struct {
	Kind, Name, Message string
}

func (e notFound) Error() string { return e.Message }

func send(ctx context.Context, c *cli.Command, method, path string, body any) (*http.Response, error) {
	var in io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		in = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(c.String("server"), "/")+path, in)
	if err != nil {
		return nil, err
	}
	if token := c.String("token"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		defer res.Body.Close()
		if res.StatusCode == http.StatusUnauthorized && c.String("token") == "" {
			return nil, errors.New("no token: set MM_TOKEN or pass --token (the server's APP_AUTH_ADMINTOKEN works)")
		}
		msg, _ := io.ReadAll(res.Body)
		var answer struct {
			Error struct {
				Message string
				Details struct{ Kind, Name string }
			}
		}
		if json.Unmarshal(msg, &answer) == nil && answer.Error.Details.Kind != "" {
			return nil, notFound{answer.Error.Details.Kind, answer.Error.Details.Name, answer.Error.Message}
		}
		return nil, fmt.Errorf("%s: %s", res.Status, bytes.TrimSpace(msg))
	}
	return res, nil
}

func call(ctx context.Context, c *cli.Command, method, path string, body any) ([]byte, error) {
	res, err := send(ctx, c, method, path, body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func show(body []byte, err error) error {
	if err != nil || len(body) == 0 {
		return err
	}
	var answer struct{ Data json.RawMessage }
	if err := json.Unmarshal(body, &answer); err != nil || answer.Data == nil {
		answer.Data = body
	}
	var out bytes.Buffer
	if err := json.Indent(&out, answer.Data, "", "  "); err != nil {
		_, err = os.Stdout.Write(body)
		return err
	}
	fmt.Println(out.String())
	return nil
}
