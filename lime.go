package main

import (
    "io/ioutil"
    "path/filepath"
    "html/template"
    "launchpad.net/goyaml"
    "regexp"
    "time"
    "log"
    "os"
    "strings"
)

type convertible struct {
    data map[string]interface{}
    content string
}

type post struct {
    convertible
    site *site
    time time.Time
    name string
    dir string
    base string
    slug string
    extract string
    output string
    categories []string
    tags []string
    published bool
}

type site struct {
    time time.Time
    config map[string]interface{}
    layouts *template.Template
    posts []*post
    source string
    dest string
    categories map[string][]*post
    tags map[string][]*post
    data map[string]interface{}
}

func validPost(f os.FileInfo) bool {
    return true
}

func (c *convertible) readYAML(base string, name string) {
    path := filepath.Join(base, name)
    re := regexp.MustCompile(`(?sm)---(\s*\n.*?\n?)^---\s*$\n?(.*)`)
    bs, err := ioutil.ReadFile(path)
    if err != nil {
        log.Fatal(err)
    }
    matches := re.FindSubmatch(bs)
    if len(matches) == 3 {
        if err := goyaml.Unmarshal(matches[1], &c.data); err != nil {
            log.Printf("Failed to parse front matter in '%s': %s", err)
        }
        c.content = strings.TrimSpace(string(matches[2]))
    } else {
        log.Printf("Could not find front matter in '%s'", path)
    }
}

func newPost(s *site, source string, dir string, name string) *post {
    p := &post{
        site: s, 
        name: name,
        dir: dir,
        base: filepath.Join(source, dir, s.config["posts"].(string)),
        categories: []string { },
        tags: []string { },
    }
    p.readYAML(p.base, name)
    return p
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
    s.layouts = new(template.Template)
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
        dir := filepath.Dir(e)
        p := newPost(s, s.source, dir, e)
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
    base := filepath.Join(s.source, s.config["layouts"].(string), "*.html")
    var err error
    if s.layouts, err = template.ParseGlob(base); err != nil {
        log.Fatal(err)
    }    
}

func (s *site) read() {
    s.readLayouts();
    s.readPosts();
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
    log.Printf("%v", s)
}