package main

import (
    "io"
    "io/ioutil"
    "path/filepath"
    "text/template"
    "launchpad.net/goyaml"
    "github.com/knieriem/markdown"
    "regexp"
    "time"
    "log"
    "os"
    "strings"
    "errors"
    "bytes"
    "bufio"
    "flag"
    "net/http"
    "fmt"
)

type data map[string]interface{}

type convertible struct {
    data data
    content string
}

type post struct {
    convertible
    site *site
    date time.Time
    name string
    base string
    slug string
    ext string
    extract string
    output string
    categories []string
    tags []string
}

type page struct {
    convertible
    site *site
    base string
    dir string
    name string
    basename string
    ext string
    output string
}

type layout struct {
    convertible
    site *site
    name string
    base string
    ext string
}

type site struct {
    time time.Time
    config map[string]interface{}
    layouts map[string]*layout
    posts []*post
    pages []*page
    source string
    dest string
    categories map[string][]*post
    tags map[string][]*post
    data data
}

type url struct {
    template string
    data data
    permalink string
}

func validPost(f os.FileInfo) bool {
    return true
}

func matchName(name string) (date string, slug string, ext string, err error) {
    re := regexp.MustCompile(`([0-9]*-[0-9]*-[0-9]*)-([a-zA-Z0-9\-]*).([a-z]*)`)
    matches := re.FindStringSubmatch(name)
    if len(matches) == 4 {
        date = matches[1]
        slug = matches[2]
        ext = matches[3]
        err = nil
    } else {
        err = errors.New("Could not match post name")
    }
    return
}

func parseDate(s string) time.Time {
    v, err := time.Parse("2006-01-02", s)
    if err != nil {
        log.Fatal(err)
    }
    return v
}

func hasYAMLHeader(path string) bool {
    f, err := os.Open(path)
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    rd := bufio.NewReader(f)
    scanner := bufio.NewScanner(rd)   
    if scanner.Scan() {
        return scanner.Text() == "---"
    }
    return false
}

func executeTemplate(content string, payload data) string {
    tmpl, err := template.New("t").Parse(content)
    if err != nil {
        log.Fatal(err)
    }
    var buffer bytes.Buffer
    wr := bufio.NewWriter(&buffer)
    tmpl.Execute(wr, payload)
    wr.Flush()
    return buffer.String()
}

var md = markdown.NewParser(&markdown.Extensions{Smart:true})
func transform(input string) string {
    var buffer bytes.Buffer
    rd := strings.NewReader(input)
    wr := bufio.NewWriter(&buffer)
    md.Markdown(rd, markdown.ToHTML(wr))
    wr.Flush()
    return buffer.String()
}

func (d data) fetch(key string, defaultVal interface{}) interface{} {
    v, ok := d[key]
    if !ok {
        return defaultVal
    }
    return v
}

func (d data) merge(other data) {
    for k, v := range other {
        d[k] = v
    }
}

func (c *convertible) readYAML(base string, name string) {
    path := filepath.Join(base, name)
    re := regexp.MustCompile(`(?sm)---(\s*\n.*?\n?)^---\s*$\n?(.*)`)
    bs, err := ioutil.ReadFile(path)
    if err != nil {
        log.Fatal(err)
    }
    matches := re.FindSubmatch(bs)
    if len(matches) == 3 { // all, front matter, body
        if err := goyaml.Unmarshal(matches[1], &c.data); err != nil {
            log.Printf("Failed to parse front matter in '%s': %s", err)
        }
        c.content = strings.TrimSpace(string(matches[2]))
    } else {
        c.content = strings.TrimSpace(string(bs))
    }
}

func newLayout(s *site, base string, name string) *layout {
    l := &layout{
        site: s,
        base: base,
        name: name,
    }
    l.process(name)
    l.readYAML(base, name)
    return l
}

func (l *layout) process(name string) {
    l.ext = filepath.Ext(name)
}

func (l *layout) render(layouts map[string]*layout, payload data) string {
    tmpl, err := template.New("t").Parse(l.content)
    if err != nil {
        log.Fatal(err)
    }
    var buffer bytes.Buffer
    wr := bufio.NewWriter(&buffer)
    err = tmpl.Execute(wr, payload)
    if err != nil {
        log.Fatal(err)
    }
    wr.Flush()
    output := buffer.String()
    name, ok := l.data["layout"].(string)
    if !ok {
        return output
    }
    parent, ok := layouts[name]
    if ok {
        payload.merge(data { "content": output })
        return parent.render(layouts, payload)
    }
    return output
}

func newPost(s *site, source string, name string) *post {
    p := &post{
        site: s, 
        name: name,
        base: filepath.Join(source, s.config["posts"].(string)),
        tags: []string { },
    }
    p.process()
    p.readYAML(p.base, name)
    v, ok := p.data["date"]
    if ok {
        p.date = parseDate(v.(string))
    }
    p.populateCategories()
    p.populateTags()
    return p
}

func (p *post) populateCategories() {
    cats, _ := p.data.fetch("categories", make([]interface{}, 0, 0)).([]interface{})
    p.categories = make([]string, 0, len(cats))
    for _, c := range cats {
        p.categories = append(p.categories, c.(string))
    }
}

func (p *post) populateTags() {
    tags, _ := p.data.fetch("tags", make([]interface{}, 0, 0)).([]interface{})
    p.tags = make([]string, 0, len(tags))
    for _, t := range tags {
        p.tags = append(p.tags, t.(string))
    }
}

func (p *post) process() {
    datestr, slug, ext, err := matchName(p.name)
    if err != nil {
        log.Fatal(err)
    }
    p.date = parseDate(datestr)
    p.slug = slug
    p.ext = ext
}

func (p *post) title() string {
    return p.data.fetch("title", p.titleizedSlug()).(string)
}

func (p *post) titleizedSlug() string {
    chunks := strings.Split(p.slug, "-")
    capitalized := make([]string, 0, len(chunks))
    for _, c := range chunks {
        c = strings.TrimSpace(c)
        if len(c) > 0 {
            capitalized = append(capitalized, strings.Title(c))
        }
    }
    return strings.Join(capitalized, " ")
}

func (p *post) published() bool {
    v, _ := p.data.fetch("published", false).(bool)
    return v
}

func (p *post) index() int {
    for i, _ := range p.site.posts {
        if p.site.posts[i] == p {
            return i
        }
    }
    return -1
}

func (p *post) next() *post {
    pos := p.index()
    if pos != -1 && pos < len(p.site.posts) - 1 {
        return p.site.posts[pos + 1]
    }
    return nil
}

func (p *post) previous() *post {
    pos := p.index()
    if pos != -1 && pos > 0 {
        return p.site.posts[pos - 1]
    }
    return nil
}

func (p *post) placeholders() data {
    return data {
        "categories": strings.Join(p.categories, "/"),
        "year": p.date.Year(),
        "month": int(p.date.Month()),
        "day": p.date.Day(),
        "title": p.slug,
    }
}

func (p *post) template() string {
    return "/{{.categories}}/{{.year}}/{{.month}}/{{.day}}/{{.title}}.html"
}

func (p *post) url() string {
    u := newUrl(p.template(), p.placeholders(), "")
    return u.String()
}

func (p *page) url() string {
    u := newUrl(p.template(), p.placeholders(), "")
    return u.String()
}

func (p *post) render(payload data) {
    payload.merge(data { "page": p.data })
    p.output = executeTemplate(p.content, payload)
    p.output = transform(p.output)
    name, ok := p.data["layout"].(string)
    if !ok {
        return
    }
    layout, ok := p.site.layouts[name]
    if !ok {
        return
    }    
    payload.merge(data { "content": p.output })
    p.output = layout.render(p.site.layouts, payload)
}

func (p *page) render(payload data) {
    payload.merge(data { "page": p.data })
    p.output = executeTemplate(p.content, payload)
    name, ok := p.data["layout"].(string)
    if !ok {
        return
    }
    layout, ok := p.site.layouts[name]
    if !ok {
        return
    }
    payload.merge(data { "content": p.output })
    p.output = layout.render(p.site.layouts, payload)
}

func (p *post) write(dir string) {
    path := filepath.Join(dir, p.url())
    _, err := os.Stat(filepath.Dir(path))
    if os.IsNotExist(err) {
        os.MkdirAll(filepath.Dir(path), 0777)
    }
    err = ioutil.WriteFile(path, []byte(p.output), 0777)
    if err != nil {
        log.Fatal(err)
    }
}

func (p *page) write(dir string) {
    path := filepath.Join(dir, p.url())
    _, err := os.Stat(filepath.Dir(path))
    if os.IsNotExist(err) {
        os.MkdirAll(filepath.Dir(path), 0777)
    }
    err = ioutil.WriteFile(path, []byte(p.output), 0777)
    if err != nil {
        log.Fatal(err)
    }
}

func (p *page) template() string {
    return "/{{.path}}/{{.basename}}{{.ext}}" 
}

func (p *page) placeholders() data {
    return data {
        "path": p.dir,
        "basename": p.basename,
        "ext": p.ext,
    }
}

func (p *page) index() bool {
    return p.basename == "index"
}

func newPage(s *site, base string, dir string, name string) *page {
    p := &page {
        site: s,
        base: base,
        dir: dir,
        name: name,
    }
    p.process(name)
    p.readYAML(filepath.Join(base, dir), name)
    return p
}

func (p *page) process(name string) {
    p.ext = filepath.Ext(name)
    p.basename = name[:len(name) - len(p.ext)]
}

func newSite(config map[string]interface{}) *site {
    source, err := filepath.Abs(config["source"].(string))
    if err != nil {
        log.Fatal(err)
    }
    dest, err := filepath.Abs(config["dest"].(string))
    if err != nil {
        log.Fatal(err)
    }
    s := &site{
        config: config,
        source: source,
        dest: dest,
    }
    s.reset()
    return s
}

func (s *site) reset() {
    s.time = time.Now()
    s.layouts = map[string]*layout { }
    s.posts = make([]*post, 0, 128)
    s.data = data { }
    s.categories = make(map[string][]*post)
    s.tags = make(map[string][]*post)
}

func (s *site) entries(subfolder string) []string {
    base := filepath.Join(s.source, subfolder)
    if err := os.Chdir(base); err != nil {
        log.Fatal(err)
    }
    entries := make([]string, 0, 256)
    visit := func(path string, f os.FileInfo, err error) error {
        if !f.IsDir() { 
            entries = append(entries, path)
        }
        return nil
    }
    if err := filepath.Walk(".", visit); err != nil {
        log.Fatal(err)
    } 
    return entries
}

func (s *site) readPosts() {
    entries := s.entries(s.config["posts"].(string))
    for _, e := range entries {
        p := newPost(s, s.source, e)
        s.addPost(p)
    }
}

func (s *site) addPost(p *post) {
    s.posts = append(s.posts, p)
    for _, c := range p.categories {
        s.categories[c] = append(s.categories[c], p)
    }
    for _, t := range p.tags {
        s.tags[t] = append(s.tags[t], p)
    }
}

func (s *site) readLayouts() {
    base := filepath.Join(s.source, s.config["layouts"].(string))
    if err := os.Chdir(base); err != nil {
        log.Fatal(err)
    }
    visit := func(path string, f os.FileInfo, err error) error {
        if !f.IsDir() {
            l := newLayout(s, base, path)
            fname := filepath.Base(path)
            ext := filepath.Ext(path)
            name := fname[:len(fname) - len(ext)]
            s.layouts[name] = l
        }
        return nil
    }
    if err := filepath.Walk(".", visit); err != nil {
        log.Fatal(err)
    }
}

func (s *site) readDirectories() {
    s.pages = make([]*page, 0, 32)
    visit := func(path string, fi os.FileInfo, err error) error {
        skip := false
        n := fi.Name()
        if strings.HasPrefix(n, ".") {
            skip = true
        }
        if strings.HasPrefix(n, "_") {
            skip = true
        }
        if strings.HasPrefix(n, "#") {
            skip = true
        }
        if strings.HasSuffix(n, "~") {
            skip = true
        }
        if fi.IsDir() {
            if skip {
                return filepath.SkipDir
            }
        } else {
            if !skip && hasYAMLHeader(path) {
                name := filepath.Base(path)
                rel, err := filepath.Rel(s.source, path)
                if err != nil {
                    log.Fatal(err)
                }
                dir := filepath.Dir(rel)
                page := newPage(s, s.source, dir, name)
                s.pages = append(s.pages, page)
            }
        }
        return nil
    }
    filepath.Walk(s.source, visit)
}

func (s *site) read() {
    s.readLayouts()
    s.readPosts()
    s.readDirectories()
}

type renderer interface {
    render(payload data)
}

// This is just a huge workaround to make it
// possible to access stuff by lowercase in the
// templates because honestly, uppercase is just
// fugly in there. 
//
// NOTE: We can't access struct fields and methods 
// if they are lowercase.
func (p *post) collectPayload() data {
    return data {
        "title": p.title(),
        "url": p.url(),
        "date": p.date,
        "slug": p.slug,
        "tags": p.tags,
        "categories": p.categories,
    }
}

func (s *site) collectPayload() data {
    posts := make([]data, 0, len(s.posts))
    for _, p := range s.posts {
        d := p.collectPayload()
        if p.next() != nil {
            d["next"] = p.next().collectPayload()
        }
        if p.previous() != nil {
            d["previous"] = p.previous().collectPayload()
        }
        d.merge(p.data) // Merge into site global data
        p.data.merge(d) // Merge into post data
        posts = append(posts, d)
    }
    payload := data {
        "site": data {
            "time": s.time,
            "source": s.source,
            "posts": posts,
        },
    }
    return payload
}

func (s *site) render() {
    payload := s.collectPayload()
    for _, p := range s.posts {
        p.render(payload)
    }
    for _, p := range s.pages {
        p.render(payload)
    }
}

type writer interface {
    write(dir string)
}

func (s *site) write() {
    dir := filepath.Join(s.config["source"].(string), s.config["dest"].(string))
    for _, p := range s.posts {
        p.write(dir)
    }
    for _, p := range s.pages {
        p.write(dir)
    }
}

func newUrl(template string, data data, permalink string) *url {
    return &url{template, data, permalink}
}

func (u *url) generate() string {
    tmpl, err := template.New("t").Parse(u.template)
    if err != nil {
        log.Fatal(err)
    }
    var buffer bytes.Buffer
    wr := bufio.NewWriter(&buffer)
    tmpl.Execute(wr, u.data)
    wr.Flush()
    return buffer.String()    
}

func (u *url) String() string {
    s := u.generate()
    re := regexp.MustCompile(`\/\/`)
    s = re.ReplaceAllString(s, "/")
    // Remove all url segments that consist solely of dots
    segments := strings.Split(s, "/")
    included := make([]string, 0, 16)
    re = regexp.MustCompile(`^\.+$`)
    for _, s := range segments {
        if re.MatchString(s) {
            continue
        }
        included = append(included, s)
    }
    return strings.Join(included, "/")
}

func generate(config data) {
    log.Printf("Generate [%s]", config["source"].(string))
    t0 := time.Now()
    s := newSite(config)
    s.read()
    s.render()
    s.write()
    dur := time.Since(t0)
    log.Printf("Done! [%v]", dur)
}

func serve(config data) {
    wd := filepath.Join(config["source"].(string), config["dest"].(string))
    err := os.Chdir(wd)
    if err != nil {
        log.Fatal(err)
    }
    h := func(w http.ResponseWriter, r *http.Request) {
        path := filepath.Join(wd, r.URL.Path)
        fi, err := os.Open(path)
        if err != nil {
            fmt.Fprintf(w, "ERROR %v", err)
            log.Printf("ERROR %v", err)
            return
        }
        defer fi.Close()

        rd := bufio.NewReader(fi)
        buf := make([]byte, 1024)
        for {
            n, err := rd.Read(buf)
            if err != nil && err != io.EOF {
                log.Fatal(err)
            }
            if n == 0 {
                break
            }
            if _, err := w.Write(buf[:n]); err != nil {
                log.Fatal(err)
            }
        }
        log.Printf("%s", path)
    }    
    addr := fmt.Sprintf(":%v", 8080)
    log.Printf("Listen [localhost%s]", addr)
    http.HandleFunc("/", h)
    http.ListenAndServe(addr, nil)
}

var doserve = flag.Bool("serve", false, "serve the site from the destination directory")

func main() {
    flag.Parse()
    wd, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }
    config := data {
        "source": wd,
        "dest": "_site",
        "posts": "_posts",
        "layouts": "_layouts",
    }
    generate(config)
    if *doserve {
        serve(config)        
    }
}