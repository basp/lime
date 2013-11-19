package main

import (
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
    name string
    base string
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

func (p *post) render(payload data) {
    p.output = transform(p.content)
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
    s.data = map[string]interface{} { "TODO": "data" }
    s.categories = make(map[string][]*post)
    s.tags = make(map[string][]*post)
}

func (s *site) entries(subfolder string) []string {
    base := filepath.Join(s.source, subfolder)
    os.Chdir(base)
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
    os.Chdir(base)
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

func (s *site) read() {
    s.readLayouts();
    s.readPosts();
}

func (s *site) render() {
    payload := data {}
    for _, p := range s.posts {
        p.render(payload)
    }
}

func (s *site) write() {
    for _, p := range s.posts {
        dir := filepath.Join(s.config["source"].(string), s.config["dest"].(string))
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
    return re.ReplaceAllString(s, "/")
}

func main() {
    wd, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }
    config := map[string]interface{} {
        "source": wd,
        "dest": "_site",
        "posts": "_posts",
        "layouts" : "_layouts",
    }
    s := newSite(config)
    s.read()
    s.render()
    s.write()
}