package manifest

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.alibaba-inc.com/force/git-repo/config"
	"code.alibaba-inc.com/force/git-repo/path"
	"github.com/jiangxin/goconfig"
	"github.com/jiangxin/multi-log"
)

// Macros for manifest
const (
	ManifestXMLFile   = "manifest.xml"
	LocalManifestFile = "local_manifest.xml"
	LocalManifestDir  = "local_manifests"
	maxRecursiveDepth = 10
)

// Manifest is for toplevel XML structure
type Manifest struct {
	XMLName        xml.Name        `xml:"manifest"`
	Notice         string          `xml:"notice,omitempty"`
	Remotes        []Remote        `xml:"remote,omitempty"`
	Default        *Default        `xml:"default,omitempty"`
	Server         *Server         `xml:"manifest-server,omitempty"`
	Projects       []Project       `xml:"project,omitempty"`
	RemoveProjects []RemoveProject `xml:"remove-project,omitempty"`
	ExtendProjects []ExtendProject `xml:"extend-project,omitempty"`
	RepoHooks      *RepoHooks      `xml:"repo-hooks,omitempty"`
	Includes       []Include       `xml:"include,omitempty"`
	SourceFile     string          `xml:"-"`
}

// Remote is for remote XML element
type Remote struct {
	Name     string `xml:"name,attr,omitempty"`
	Alias    string `xml:"alias,attr,omitempty"`
	Fetch    string `xml:"fetch,attr,omitempty"`
	PushURL  string `xml:"pushurl,attr,omitempty"`
	Review   string `xml:"review,attr,omitempty"`
	Revision string `xml:"revision,attr,omitempty"`
}

// Default is for default XML element
type Default struct {
	Remote     string `xml:"remote,attr,omitempty"`
	Revision   string `xml:"revision,attr,omitempty"`
	DestBranch string `xml:"dest-branch,attr,omitempty"`
	Upstream   string `xml:"upstream,attr,omitempty"`
	SyncJ      string `xml:"sync-j,attr,omitempty"`
	SyncC      string `xml:"sync-c,attr,omitempty"`
	SyncS      string `xml:"sync-s,attr,omitempty"`
	SyncTags   string `xml:"sync-tags,attr,omitempty"`
}

// Server is for manifest-server XML element
type Server struct {
	URL string `xml:"url,attr,omitempty"`
}

// Project is for project XML element
type Project struct {
	Annotations []Annotation `xml:"annotation,omitempty"`
	Projects    []Project    `xml:"project,omitempty"`
	CopyFiles   []CopyFile   `xml:"copyfile,omitempty"`
	LinkFiles   []LinkFile   `xml:"linkfile,omitempty"`

	Name       string `xml:"name,attr,omitempty"`
	Path       string `xml:"path,attr,omitempty"`
	Remote     string `xml:"remote,attr,omitempty"`
	Revision   string `xml:"revision,attr,omitempty"`
	DestBranch string `xml:"dest-branch,attr,omitempty"`
	Groups     string `xml:"groups,attr,omitempty"`
	SyncC      string `xml:"sync-c,attr,omitempty"`
	SyncS      string `xml:"sync-s,attr,omitempty"`
	SyncTags   string `xml:"sync-tags,attr,omitempty"`
	Upstream   string `xml:"upstream,attr,omitempty"`
	CloneDepth string `xml:"clone-depth,attr,omitempty"`
	ForcePath  string `xml:"force-path,attr,omitempty"`
}

// Annotation is for annotation XML element
type Annotation struct {
	Name  string `xml:"name,attr,omitempty"`
	Value string `xml:"value,attr,omitempty"`
	Keep  string `xml:"keep,attr,omitempty"`
}

// CopyFile is for copyfile XML element
type CopyFile struct {
	Src  string `xml:"src,attr,omitempty"`
	Dest string `xml:"dest,attr,omitempty"`
}

// LinkFile is for linkfile XML element
type LinkFile struct {
	Src  string `xml:"src,attr,omitempty"`
	Dest string `xml:"dest,attr,omitempty"`
}

// ExtendProject is for extend-project XML element
type ExtendProject struct {
	Name     string `xml:"name,attr,omitempty"`
	Path     string `xml:"path,attr,omitempty"`
	Groups   string `xml:"groups,attr,omitempty"`
	Revision string `xml:"revision,attr,omitempty"`
}

// RemoveProject is for remove-project XML element
type RemoveProject struct {
	Name string `xml:"name,attr,omitempty"`
}

// RepoHooks is for repo-hooks XML element
type RepoHooks struct {
	InProject   string `xml:"in-project,attr,omitempty"`
	EnabledList string `xml:"enabled-list,attr,omitempty"`
}

// Include is for include XML element
type Include struct {
	Name string `xml:"name,attr,omitempty"`
}

// AllProjects returns proejcts of a project recursively
func (v *Project) AllProjects(pDir string) []Project {
	var project Project
	projects := []Project{}
	if pDir != "" {
		v.Path = filepath.Join(pDir, v.Path)
	}
	// remove field: Projects
	if len(v.Projects) > 0 {
		project = Project{
			Annotations: v.Annotations,
			CopyFiles:   v.CopyFiles,
			LinkFiles:   v.LinkFiles,
			Name:        v.Name,
			Path:        v.Path,
			Remote:      v.Remote,
			Revision:    v.Revision,
			DestBranch:  v.DestBranch,
			Groups:      v.Groups,
			SyncC:       v.SyncC,
			SyncS:       v.SyncS,
			SyncTags:    v.SyncTags,
			Upstream:    v.Upstream,
			CloneDepth:  v.CloneDepth,
			ForcePath:   v.ForcePath,
		}
		projects = append(projects, project)
	} else {
		projects = append(projects, *v)
	}

	for _, p := range v.Projects {
		projects = append(projects, p.AllProjects(v.Path)...)
	}
	return projects
}

// AllProjects returns all projects of manifest
func (v *Manifest) AllProjects() []Project {
	projects := []Project{}
	for _, p := range v.Projects {
		projects = append(projects, p.AllProjects("")...)
	}
	return projects
}

// Merge will merge another manifest to self
func (v *Manifest) Merge(m *Manifest) error {
	if m.Notice != "" {
		if v.Notice == "" {
			v.Notice = m.Notice
		} else {
			return fmt.Errorf("duplicate notice in %s", m.SourceFile)
		}
	}

	for _, r1 := range m.Remotes {
		found := false
		for _, r2 := range v.Remotes {
			if r1.Name == r2.Name {
				if r1 != r2 {
					return fmt.Errorf("duplicate remote in %s", m.SourceFile)
				}
				found = true
				break
			}
		}
		if !found {
			v.Remotes = append(v.Remotes, r1)
		}
	}

	if m.Default != nil {
		if v.Default != nil {
			if v.Default != m.Default {
				return fmt.Errorf("duplicate default in %s", m.SourceFile)
			}
		} else {
			v.Default = m.Default
		}
	}

	if m.Server != nil {
		if v.Server != nil {
			if v.Server != m.Server {
				return fmt.Errorf("duplicate manifest-server in %s", m.SourceFile)
			}
		} else {
			v.Server = m.Server
		}
	}

	realPath := make(map[string]bool)
	for _, p := range v.AllProjects() {
		if realPath[p.Path] {
			return fmt.Errorf("duplicate path for project '%s' in '%s'",
				p.Path,
				v.SourceFile)
		}
		realPath[p.Path] = true
	}
	for _, p := range m.AllProjects() {
		if realPath[p.Path] {
			return fmt.Errorf("duplicate path for project '%s' in '%s'",
				p.Path,
				m.SourceFile)
		}
		v.Projects = append(v.Projects, p)
		realPath[p.Path] = true
	}

	rmPath := make(map[string]bool)
	for _, r := range m.RemoveProjects {
		rmPath[r.Name] = true
	}
	ps := []Project{}
	for _, p := range v.AllProjects() {
		if rmPath[p.Name] {
			realPath[p.Path] = false
		} else {
			ps = append(ps, p)
		}
	}
	v.Projects = ps

	extPath := make(map[string]ExtendProject)
	for _, p := range m.ExtendProjects {
		extPath[p.Name] = p
	}
	for i, p := range v.Projects {
		if p2, ok := extPath[p.Name]; ok {
			if p2.Path == p.Path {
				if p.Groups == "" {
					v.Projects[i].Groups = p2.Groups
				} else if p2.Groups != "" {
					groups := []string{}
					groups = append(groups, strings.Split(p.Groups, ",")...)
					groups = append(groups, strings.Split(p2.Groups, ",")...)
					v.Projects[i].Groups = strings.Join(groups, ",")
				}
				if p2.Revision != "" {
					v.Projects[i].Revision = p2.Revision
				}
			}
		}
	}

	// m.RepoHooks

	return nil
}

func unmarshal(file string) (*Manifest, error) {
	manifest := Manifest{}
	if _, err := os.Stat(file); err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read manifest file '%s': %s", file, err)
	}

	err = xml.Unmarshal(buf, &manifest)
	if err != nil {
		return nil, fmt.Errorf("fail to parse manifest file '%s': %s", file, err)
	}
	return &manifest, nil
}

func parseXML(file string, depth int) ([]*Manifest, error) {
	ms := []*Manifest{}

	m, err := unmarshal(file)
	if err != nil {
		return ms, err
	}
	if m == nil {
		return ms, nil
	}
	m.SourceFile = file
	ms = append(ms, m)

	for _, i := range m.Includes {
		f := path.AbsJoin(filepath.Dir(file), i.Name)

		if depth > maxRecursiveDepth {
			return nil, fmt.Errorf("exceeded maximum include depth (%d) while including\n"+
				"\t%s\n"+
				"from"+
				"\t%s\n"+
				"This might be due to circular includes",
				maxRecursiveDepth,
				f,
				file)
		}

		subMs, err := parseXML(f, depth+1)
		if err != nil {
			return ms, err
		}
		ms = append(ms, subMs...)
	}

	return ms, nil
}

func mergeManifests(ms []*Manifest) (*Manifest, error) {
	manifest := &Manifest{}
	for _, m := range ms {
		err := manifest.Merge(m)
		if err != nil {
			return nil, err
		}
	}
	return manifest, nil
}

// Load will load and parse manifest XML file
func Load(repoDir string) (*Manifest, error) {
	var (
		file      string
		dir       string
		err       error
		manifests = []*Manifest{}
	)

	file = filepath.Join(repoDir, ManifestXMLFile)
	if _, err = os.Stat(file); err != nil {
		defaultXML := ""
		manifestsDir := filepath.Join(repoDir, "manifests")
		cfg, err := goconfig.LoadDir(manifestsDir, false)
		if err != nil {
			return nil, fmt.Errorf("fail to read config from %s: %s", manifestsDir, err)
		}
		if cfg != nil {
			defaultXML = cfg.Get(config.RepoDefaultManifestKey)
		}
		if defaultXML == "" {
			defaultXML = config.RepoDefaultManifestXML
		}
		file = filepath.Join(manifestsDir, defaultXML)
	}

	// Ignore uninitialized repo
	if _, err := os.Stat(file); err != nil {
		return nil, nil
	}

	ms, err := parseXML(file, 1)
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, ms...)

	// load LocalManifestFile (obsolete)
	files := []string{}
	file = filepath.Join(repoDir, LocalManifestFile)
	dir = filepath.Join(repoDir, LocalManifestDir)
	if _, err = os.Stat(file); err == nil {
		log.Warnf("%s is deprecated; put local manifests in `%s` instead", file, dir)
		files = append(files, file)
	}

	// load xml files in LocalManifestDir
	if _, err = os.Stat(dir); err == nil {
		filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if dir == name {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(name, ".xml") {
				files = append(files, name)
			}
			return nil
		})
	}

	for _, file = range files {
		ms, err := parseXML(file, 1)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, ms...)
	}

	return mergeManifests(manifests)
}