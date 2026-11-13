package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func qs(kv ...string) url.Values { // query from key,value pairs; empty values skipped
	q := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] != "" {
			q.Set(kv[i], kv[i+1])
		}
	}
	return q
}

func oneArg(c *cobra.Command) { c.Args = cobra.ExactArgs(1) }

// ---------- auth / config ----------

func readSecret(prompt string) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(errOut, prompt)
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(errOut)
		return strings.TrimSpace(string(b)), err
	}
	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", fail(exitUsage, "expected two lines on stdin: user key, then sender key")
	}
	return strings.TrimSpace(line), nil
}

func printObject(pairs [][2]string) error {
	f, err := pickFormat(g.asJSON, g.asNDJSON)
	if err != nil {
		return err
	}
	if f == fmtTable {
		for _, p := range pairs {
			fmt.Fprintf(out, "%s: %s\n", p[0], p[1])
		}
		return nil
	}
	m := map[string]string{}
	for _, p := range pairs {
		m[p[0]] = p[1]
	}
	b, _ := json.Marshal(m)
	return emit(f, b, "", nil)
}

func authCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Manage credentials (stored in the OS keychain)"}
	login := &cobra.Command{
		Use:   "login",
		Short: "Store the user and sender keys in the OS keychain for --env",
		Long: "Prompts for the keys without echo, or reads two lines from stdin (user key, then sender key) when piped.\n" +
			"Keys are never accepted as arguments, so they stay out of shell history and process lists.\n" +
			"  printf '%s\\n%s\\n' \"$USER_KEY\" \"$SENDER_KEY\" | konsult auth login",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			u, err := readSecret("User key: ")
			if err != nil {
				return err
			}
			s, err := readSecret("Sender key: ")
			if err != nil {
				return err
			}
			if u == "" || s == "" {
				return fail(exitUsage, "both keys are required")
			}
			env := strings.ToLower(g.env)
			if _, err := resolveBase(env, ""); err != nil {
				return err
			}
			if err := saveCreds(env, creds{u, s}); err != nil {
				return fail(exitGeneral, "cannot write to the OS keychain: %v", err)
			}
			fmt.Fprintf(errOut, "credentials for %s saved in the OS keychain\n", env)
			return nil
		},
	}
	logout := &cobra.Command{
		Use: "logout", Short: "Remove stored credentials for --env from the OS keychain", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			if err := deleteCreds(strings.ToLower(g.env)); err != nil {
				return fail(exitGeneral, "%v", err)
			}
			fmt.Fprintf(errOut, "credentials for %s removed\n", strings.ToLower(g.env))
			return nil
		},
	}
	var verify bool
	status := &cobra.Command{
		Use: "status", Short: "Show environment and credential source (never the values)", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			base, err := resolveBase(g.env, g.baseURL)
			if err != nil {
				return err
			}
			pairs := [][2]string{{"environment", strings.ToLower(g.env)}, {"baseUrl", base}, {"credentials", describeCreds(strings.ToLower(g.env))}}
			if verify {
				c, err := getClient()
				if err != nil {
					return err
				}
				resp, err := c.Do("GET", "/consultant", nil, nil) // body discarded: it contains personal data
				if err != nil {
					return err
				}
				resp.Body.Close()
				pairs = append(pairs, [2]string{"verified", "true"})
			}
			return printObject(pairs)
		},
	}
	status.Flags().BoolVar(&verify, "verify", false, "check the credentials with one read request (GET /consultant, result discarded)")
	cmd.AddCommand(login, logout, status)
	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Show effective settings"}
	cmd.AddCommand(&cobra.Command{
		Use: "show", Short: "Show effective environment, base URL and credential source", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			base, err := resolveBase(g.env, g.baseURL)
			if err != nil {
				return err
			}
			return printObject([][2]string{{"environment", strings.ToLower(g.env)}, {"baseUrl", base}, {"credentials", describeCreds(strings.ToLower(g.env))}, {"timeout", g.timeout.String()}})
		},
	})
	return cmd
}

// ---------- market ----------

func marketCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "market", Short: "Marketplace assignments (needs Premium Sales; export quota applies)"}
	var text, match, published, file string
	var profile int
	var remote, onsite, hybrid bool
	search := &cobra.Command{
		Use:   "search",
		Short: "Search the marketplace (POST /market/search); may consume export quota",
		Long:  "Searches the marketplace. Results count against your export quota (documented: 100 per 30 days under Premium Sales).\nNo pagination or polling is done; one call is one request. A saved profile id overrides other filters except publishedAfter.\nUse --file for the full documented filter body.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			body := map[string]any{}
			if file != "" {
				var err error
				if body, err = readJSONFile(file); err != nil {
					return err
				}
			}
			set := func(k string, v any) { body[k] = v }
			if text != "" {
				set("text", text)
			}
			if match != "" {
				if m := strings.ToLower(match); m != "or" && m != "and" && m != "exact" {
					return fail(exitUsage, "--match must be or, and or exact")
				}
				set("textMatchType", strings.ToLower(match))
			}
			if c.Flags().Changed("profile") {
				set("id", profile)
			}
			if published != "" {
				set("publishedAfter", published)
			}
			for _, b := range []struct {
				n string
				v bool
				k string
			}{{"remote", remote, "allowRemote"}, {"onsite", onsite, "allowOnsite"}, {"hybrid", hybrid, "allowHybrid"}} {
				if c.Flags().Changed(b.n) {
					set(b.k, b.v)
				}
			}
			b, err := read("POST", "/market/search", nil, body, "results", []string{"id", "title", "company", "publishedDate", "location"})
			warnQuota(b)
			return err
		},
	}
	fl := search.Flags()
	fl.StringVar(&text, "text", "", "free-text query")
	fl.StringVar(&match, "match", "", "text match type: or, and, exact")
	fl.IntVar(&profile, "profile", 0, "saved search profile id")
	fl.StringVar(&published, "published-after", "", "only assignments published after this date")
	fl.BoolVar(&remote, "remote", false, "allowRemote filter")
	fl.BoolVar(&onsite, "onsite", false, "allowOnsite filter")
	fl.BoolVar(&hybrid, "hybrid", false, "allowHybrid filter")
	fl.StringVar(&file, "file", "", "JSON file with the full filter body (- for stdin)")
	get := &cobra.Command{
		Use: "get ID [ID...]", Short: "Get marketplace assignments by id (max 100); may consume export quota",
		Args: cobra.RangeArgs(1, 100),
		RunE: func(_ *cobra.Command, a []string) error {
			b, err := read("GET", "/market", qs("id", strings.Join(a, ",")), nil, "results", []string{"id", "title", "company", "publishedDate", "location"})
			warnQuota(b)
			return err
		},
	}
	cmd.AddCommand(search, get)
	return cmd
}

// warnQuota reports export-quota fields on stderr; the response shape is verified only against docs.
func warnQuota(b []byte) {
	var q struct {
		Exported *float64 `json:"exportedInPeriod"`
		Volume   *float64 `json:"volume"`
		Period   any      `json:"period"`
	}
	if json.Unmarshal(b, &q) != nil || q.Exported == nil {
		return
	}
	s := fmt.Sprintf("market quota: exportedInPeriod=%v", *q.Exported)
	if q.Volume != nil {
		s += fmt.Sprintf(" volume=%v", *q.Volume)
		if *q.Volume > 0 && *q.Exported >= 0.8**q.Volume {
			s += "  WARNING: over 80% of the export quota is used"
		}
	}
	if q.Period != nil {
		pj, _ := json.Marshal(q.Period)
		s += " period=" + string(pj)
	}
	fmt.Fprintln(errOut, s)
}

// ---------- assignments ----------

var assignmentCols = []string{"id", "title", "publicationEndDate", "assignmentStartDate", "assignmentEndDate", "created"}

func assignmentsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "assignments", Short: "Your organisation's own assignments (not the marketplace)"}

	var id, from, to, phrase, status string
	var closed bool
	list := &cobra.Command{
		Use: "list", Short: "List own assignments", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			q := qs("id", id, "startdate", from, "enddate", to, "phrase", phrase, "status", status)
			if closed {
				q.Set("includeclosed", "true")
			}
			_, err := read("GET", "/assignment", q, nil, "", assignmentCols)
			return err
		},
	}
	list.Flags().StringVar(&id, "id", "", "assignment id")
	list.Flags().StringVar(&from, "from", "", "start date filter (startdate)")
	list.Flags().StringVar(&to, "to", "", "end date filter (enddate)")
	list.Flags().StringVar(&phrase, "phrase", "", "search phrase")
	list.Flags().StringVar(&status, "status", "", "status filter")
	list.Flags().BoolVar(&closed, "include-closed", false, "include closed assignments")

	get := &cobra.Command{Use: "get ID", Short: "Get one own assignment", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			_, err := read("GET", "/assignment", qs("id", a[0]), nil, "", assignmentCols)
			return err
		}}

	var cf, uf string
	cm, um := &mutFlags{}, &mutFlags{}
	create := &cobra.Command{Use: "create", Short: "Create an assignment from a JSON file", Args: cobra.NoArgs,
		Long: "Creates an assignment. Omitting the publication date is expected to create a draft (verified in the sandbox only if noted in docs/sandbox-findings.md).\nSetting a publication date or noDelay publishes it and may be externally visible or chargeable.",
		RunE: func(*cobra.Command, []string) error {
			body, err := readJSONFile(cf)
			if err != nil {
				return err
			}
			return cm.run(mutation{"POST", "/assignment", nil, body, "Creates an assignment. If it has a publication date or noDelay it becomes visible on the marketplace."})
		}}
	create.Flags().StringVar(&cf, "file", "", "assignment JSON (- for stdin)")
	create.MarkFlagRequired("file")
	cm.bind(create)

	update := &cobra.Command{Use: "update ID", Short: "Update an assignment from a JSON file (sets id from ID)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			body, err := readJSONFile(uf)
			if err != nil {
				return err
			}
			body["id"] = idVal(a[0])
			return um.run(mutation{"PUT", "/assignment", nil, body, "Replaces the assignment fields; the API expects all required fields in the body."})
		}}
	update.Flags().StringVar(&uf, "file", "", "assignment JSON (- for stdin)")
	update.MarkFlagRequired("file")
	um.bind(update)

	cmd.AddCommand(list, get, create, update)
	idAction := func(use, short, method, path, warn string, inQuery, inBody bool) *cobra.Command {
		mf := &mutFlags{}
		c := &cobra.Command{Use: use + " ID", Short: short, Args: cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, a []string) error {
				mu := mutation{method: method, path: path, warn: warn}
				if inQuery {
					mu.q = qs("id", a[0])
				}
				if inBody {
					mu.body = map[string]any{"id": idVal(a[0])}
				}
				return mf.run(mu)
			}}
		mf.bind(c)
		return c
	}
	cmd.AddCommand(
		idAction("unpublish", "Unpublish an assignment", "POST", "/assignment/unpublish", "Removes the assignment from the marketplace.", true, false),
		idAction("close", "Close an assignment (rejects pending applications and emails applicants)", "POST", "/assignment/close",
			"CLOSE unpublishes the assignment, automatically REJECTS all pending applications and EMAILS the applicants. Cannot be undone.", true, false),
		idAction("delete", "Delete a never-published assignment", "DELETE", "/assignment/delete", "Permanently deletes the assignment (only never-published ones can be deleted).", true, false),
	)

	// tags
	tags := &cobra.Command{Use: "tags", Short: "Suggested tags"}
	var ttext string
	suggest := &cobra.Command{Use: "suggest", Short: "Suggest tags for a text (read-only POST)", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			_, err := read("POST", "/assignment/suggestedtags", nil, map[string]string{"text": ttext}, "", nil)
			return err
		}}
	suggest.Flags().StringVar(&ttext, "text", "", "text to analyse")
	suggest.MarkFlagRequired("text")
	tags.AddCommand(suggest)
	cmd.AddCommand(tags)

	// files
	files := &cobra.Command{Use: "files", Short: "Assignment attachments"}
	am := &mutFlags{}
	add := &cobra.Command{Use: "add ID LOCALFILE", Short: "Upload an attachment (replaces a file with the same name)", Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, a []string) error {
			data, err := os.ReadFile(a[1])
			if err != nil {
				return fail(exitUsage, "%v", err)
			}
			body := map[string]any{"id": idVal(a[0]), "fileName": filepath.Base(a[1]), "data": base64.StdEncoding.EncodeToString(data)}
			return am.run(mutation{"POST", "/assignmentfile", nil, body, "Uploads the file; an existing attachment with the same name is replaced."})
		}}
	am.bind(add)
	var out1 string
	var force1 bool
	fget := &cobra.Command{Use: "get ID FILENAME", Short: "Download an attachment", Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, a []string) error {
			return download("/assignmentfile", qs("id", a[0], "fileName", a[1]), out1, force1)
		}}
	fget.Flags().StringVarP(&out1, "output", "o", "", "destination file (- for stdout)")
	fget.Flags().BoolVar(&force1, "force", false, "overwrite an existing file")
	dm := &mutFlags{}
	fdel := &cobra.Command{Use: "delete ID FILENAME", Short: "Delete an attachment", Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, a []string) error {
			return dm.run(mutation{"DELETE", "/assignmentfile", nil, map[string]any{"id": idVal(a[0]), "fileName": a[1]}, "Permanently deletes the attachment. (DELETE with a JSON body: unverified for every server.)"})
		}}
	dm.bind(fdel)
	files.AddCommand(add, fget, fdel)
	cmd.AddCommand(files)
	return cmd
}

// ---------- consultants ----------

func consultantsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "consultants", Short: "Your organisation's consultant profiles (contain personal data)"}
	cols := []string{"id", "firstName", "lastName", "email"}
	var query string
	list := &cobra.Command{Use: "list", Short: "List consultants", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			_, err := read("GET", "/consultant", qs("query", query), nil, "", cols)
			return err
		}}
	list.Flags().StringVar(&query, "query", "", "search text")
	get := &cobra.Command{Use: "get ID", Short: "Get a full profile (numeric id or email)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			_, err := read("GET", "/consultant", qs("id", a[0]), nil, "", cols)
			return err
		}}
	var cf, uf string
	cm, um := &mutFlags{}, &mutFlags{}
	create := &cobra.Command{Use: "create", Short: "Create a consultant from a JSON file", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			body, err := readJSONFile(cf)
			if err != nil {
				return err
			}
			return cm.run(mutation{"POST", "/consultant", nil, body, "Creates an employee with the consultant role."})
		}}
	create.Flags().StringVar(&cf, "file", "", "consultant JSON (- for stdin)")
	create.MarkFlagRequired("file")
	cm.bind(create)
	update := &cobra.Command{Use: "update ID", Short: "Update a consultant from a JSON file (sets id from ID)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			body, err := readJSONFile(uf)
			if err != nil {
				return err
			}
			body["id"] = idVal(a[0])
			return um.run(mutation{"PUT", "/consultant", nil, body, "Updates the consultant profile."})
		}}
	update.Flags().StringVar(&uf, "file", "", "consultant JSON (- for stdin)")
	update.MarkFlagRequired("file")
	um.bind(update)

	dm := &mutFlags{}
	del := &cobra.Command{Use: "delete ID", Short: "Delete a consultant", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			return dm.run(mutation{"DELETE", "/consultant", qs("id", a[0]), nil, "Permanently deletes the consultant."})
		}}
	dm.bind(del)

	var sActive, bActive, charge bool
	sm, bm := &mutFlags{}, &mutFlags{}
	share := &cobra.Command{Use: "share ID", Short: "Share (or unshare with --active=false) a profile on the marketplace", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			return sm.run(mutation{"POST", "/consultant/share", qs("id", a[0], "active", strconv.FormatBool(sActive)), nil,
				"Sharing makes the profile searchable by ALL companies using Brainville."})
		}}
	share.Flags().BoolVar(&sActive, "active", true, "set shared status (--active=false to unshare)")
	sm.bind(share)
	boost := &cobra.Command{Use: "boost ID", Short: "Boost a profile (PAID feature)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			if bActive && !charge && !bm.dryRun {
				return fail(exitAborted, "boost is a paid feature: add --accept-charge to confirm you have approved the cost (or use --dry-run)")
			}
			return bm.run(mutation{"POST", "/consultant/boost", qs("id", a[0], "active", strconv.FormatBool(bActive)), nil,
				"BOOST is a PAID feature (needs a registered card) and makes the profile highly visible, incl. daily emails to subscribers."})
		}}
	boost.Flags().BoolVar(&bActive, "active", true, "set boosted status (--active=false to stop)")
	boost.Flags().BoolVar(&charge, "accept-charge", false, "acknowledge that boosting may incur a charge")
	bm.bind(boost)
	cmd.AddCommand(list, get, create, update, del, share, boost)
	return cmd
}

// ---------- applications ----------

var (
	appStatuses   = []string{"Pending", "Accepted", "Rejected", "Negotiating", "Approved", "RequestAdditions"}
	rejectReasons = []string{"PriceHigh", "Overqualified", "Underqualified", "NotMatching", "Other"}
)

func oneOf(v string, set []string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func applicationsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "applications", Short: "Incoming applications (needs Premium Buyer or Applications licence; contain personal data)"}
	cols := []string{"id", "assignmentId", "assignmentTitle", "status", "sentDate", "resourceName"}
	var id, assignment, requestor, status, from, to string
	list := &cobra.Command{Use: "list", Short: "List applications", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			_, err := read("GET", "/application", qs("id", id, "assignmentId", assignment, "requestorId", requestor, "status", status, "startDate", from, "endDate", to), nil, "", cols)
			return err
		}}
	fl := list.Flags()
	fl.StringVar(&id, "id", "", "application id")
	fl.StringVar(&assignment, "assignment", "", "assignment id")
	fl.StringVar(&requestor, "requestor", "", "requestor id")
	fl.StringVar(&status, "status", "", "status filter")
	fl.StringVar(&from, "from", "", "startDate")
	fl.StringVar(&to, "to", "", "endDate")
	get := &cobra.Command{Use: "get ID", Short: "Get one application", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			_, err := read("GET", "/application", qs("id", a[0]), nil, "", cols)
			return err
		}}

	var set, reply, reason string
	sm := &mutFlags{}
	st := &cobra.Command{Use: "status ID --set STATUS", Short: "Change an application's status (visible to the applicant)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			if !oneOf(set, appStatuses) {
				return fail(exitUsage, "--set must be one of %s", strings.Join(appStatuses, ", "))
			}
			body := map[string]any{"id": idVal(a[0]), "status": set}
			if reply != "" {
				body["reply"] = reply
			}
			if reason != "" {
				if !oneOf(reason, rejectReasons) {
					return fail(exitUsage, "--reject-reason must be one of %s", strings.Join(rejectReasons, ", "))
				}
				body["rejectReason"] = reason
			}
			return sm.run(mutation{"PUT", "/application", nil, body, "Changes the application status; the applicant may be notified (reply text is sent)."})
		}}
	st.Flags().StringVar(&set, "set", "", "new status: "+strings.Join(appStatuses, ", "))
	st.Flags().StringVar(&reply, "reply", "", "reply text")
	st.Flags().StringVar(&reason, "reject-reason", "", "reason when Rejected: "+strings.Join(rejectReasons, ", "))
	st.MarkFlagRequired("set")
	sm.bind(st)

	var o1, o2, fname string
	var idx int
	var f1, f2 bool
	resume := &cobra.Command{Use: "resume ID", Short: "Download the applicant's CV (PDF)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, a []string) error {
			return download("/application/resume", qs("id", a[0]), o1, f1)
		}}
	resume.Flags().StringVarP(&o1, "output", "o", "", "destination file (- for stdout)")
	resume.Flags().BoolVar(&f1, "force", false, "overwrite an existing file")
	files := &cobra.Command{Use: "files", Short: "Application attachments"}
	fget := &cobra.Command{Use: "get ID", Short: "Download an application attachment by --index or --name", Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, a []string) error {
			q := qs("id", a[0], "filename", fname)
			if c.Flags().Changed("index") {
				q.Set("fileindex", strconv.Itoa(idx))
			}
			if len(q) < 2 {
				return fail(exitUsage, "give --index or --name")
			}
			return download("/application/file", q, o2, f2)
		}}
	fget.Flags().IntVar(&idx, "index", 0, "file index as listed in the application's file property")
	fget.Flags().StringVar(&fname, "name", "", "file name")
	fget.Flags().StringVarP(&o2, "output", "o", "", "destination file (- for stdout)")
	fget.Flags().BoolVar(&f2, "force", false, "overwrite an existing file")
	files.AddCommand(fget)
	cmd.AddCommand(list, get, st, resume, files)
	return cmd
}

// ---------- api request (restricted escape hatch) ----------

func apiCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "api", Short: "Escape hatch for documented operations not yet modelled"}
	cmd.AddCommand(&cobra.Command{
		Use: "request GET /v2/PATH", Short: "Send a GET to a relative /v2/ path on the configured host (write methods are disabled)",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, a []string) error {
			if !strings.EqualFold(a[0], "GET") {
				return fail(exitUsage, "only GET is allowed; use the dedicated commands for changes")
			}
			p := a[1]
			if !strings.HasPrefix(p, "/v2/") || strings.Contains(p, "..") || strings.Contains(p, "//") || strings.Contains(p, "://") || strings.ContainsAny(p, "\\#") {
				return fail(exitUsage, "path must be a relative /v2/... path")
			}
			u, err := url.Parse(strings.TrimPrefix(p, "/v2"))
			if err != nil || u.Host != "" {
				return fail(exitUsage, "invalid path")
			}
			_, err = read("GET", u.Path, u.Query(), nil, "", []string{"id"})
			return err
		},
	})
	return cmd
}
