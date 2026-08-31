package iso

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// node is a file or directory in the image tree.
type node struct {
	name     string
	isDir    bool
	srcPath  string
	data     []byte
	size     int64
	modTime  time.Time
	children []*node

	feBlock   uint32
	dataLB    uint32
	dataLen   uint32
	uniqueID  uint64
	absSector uint64
	isoName   string
}

func (n *node) find(name string) *node {
	for _, c := range n.children {
		if strings.EqualFold(c.name, name) {
			return c
		}
	}
	return nil
}

func (n *node) mkdir(name string, mt time.Time) *node {
	if c := n.find(name); c != nil {
		return c
	}
	c := &node{name: name, isDir: true, modTime: mt}
	n.children = append(n.children, c)
	return c
}

func buildTree(srcDir string, extras []Extra, gens []genFile, now time.Time) (*node, error) {
	st, err := os.Stat(srcDir)
	if err != nil {
		return nil, fmt.Errorf("iso: stat %s: %w", srcDir, err)
	}
	root := &node{name: "", isDir: true, modTime: st.ModTime()}
	video := filepath.Join(srcDir, "VIDEO_TS")
	if _, err := os.Stat(video); err != nil {
		return nil, fmt.Errorf("iso: VIDEO_TS/VIDEO_TS.IFO is required")
	}
	if _, err := os.Stat(filepath.Join(video, "VIDEO_TS.IFO")); err != nil {
		return nil, fmt.Errorf("iso: VIDEO_TS/VIDEO_TS.IFO is required")
	}
	if err := addDirChildren(root.mkdir("VIDEO_TS", now), video, true); err != nil {
		return nil, err
	}
	sortVideoTS(root.find("VIDEO_TS"))

	audio := filepath.Join(srcDir, "AUDIO_TS")
	if st, err := os.Stat(audio); err == nil && st.IsDir() {
		if err := addDirChildren(root.mkdir("AUDIO_TS", st.ModTime()), audio, true); err != nil {
			return nil, err
		}
	} else {
		root.mkdir("AUDIO_TS", now)
	}

	for _, e := range extras {
		if err := addExtra(root, e, now); err != nil {
			return nil, err
		}
	}
	for _, g := range gens {
		if root.find(g.name) != nil {
			return nil, fmt.Errorf("iso: generated %s already exists", g.name)
		}
		root.children = append(root.children, &node{
			name:    g.name,
			size:    int64(len(g.data)),
			data:    g.data,
			modTime: g.modTime,
		})
	}
	return root, nil
}

func addDirChildren(n *node, dir string, recursive bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("iso: read %s: %w", dir, err)
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		info, err := e.Info()
		if err != nil {
			return err
		}
		if e.IsDir() {
			if !recursive {
				continue
			}
			c := n.mkdir(e.Name(), info.ModTime())
			if err := addDirChildren(c, full, true); err != nil {
				return err
			}
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".iso" || ext == ".cue" {
			continue
		}
		n.children = append(n.children, &node{
			name:    e.Name(),
			srcPath: full,
			size:    info.Size(),
			modTime: info.ModTime(),
		})
	}
	return nil
}

func addExtra(root *node, e Extra, now time.Time) error {
	st, err := os.Stat(e.Host)
	if err != nil {
		return err
	}
	parts := strings.Split(e.Disc, "/")
	parent := root
	for i, p := range parts {
		last := i == len(parts)-1
		if last && !st.IsDir() {
			if parent.find(p) != nil {
				return fmt.Errorf("iso: duplicate disc path %s", e.Disc)
			}
			parent.children = append(parent.children, &node{
				name:    p,
				srcPath: e.Host,
				size:    st.Size(),
				modTime: st.ModTime(),
			})
			return nil
		}
		if last && st.IsDir() {
			d := parent.mkdir(p, st.ModTime())
			return addDirChildren(d, e.Host, e.Recursive)
		}
		parent = parent.mkdir(p, now)
	}
	return nil
}

func sortVideoTS(n *node) {
	if n == nil {
		return
	}
	sort.SliceStable(n.children, func(i, j int) bool {
		return videoRank(n.children[i].name) < videoRank(n.children[j].name)
	})
}

func videoRank(name string) string {
	u := strings.ToUpper(name)
	switch u {
	case "VIDEO_TS.IFO":
		return "0_0"
	case "VIDEO_TS.VOB":
		return "0_1"
	case "VIDEO_TS.BUP":
		return "0_2"
	}
	if strings.HasPrefix(u, "VTS_") && len(u) >= 8 {
		set := u[4:6]
		rest := u[6:]
		kind := "9"
		switch {
		case strings.HasSuffix(rest, "_0.IFO"):
			kind = "0"
		case strings.HasSuffix(rest, "_0.VOB"):
			kind = "1"
		case strings.HasSuffix(rest, "_0.BUP"):
			kind = "2"
		default:
			kind = "3" + rest
		}
		return "1_" + set + "_" + kind
	}
	return "2_" + u
}

func walkFiles(n *node, prefix string, fn func(rel string, n *node)) {
	for _, c := range n.children {
		rel := c.name
		if prefix != "" {
			rel = prefix + "/" + c.name
		}
		if c.isDir {
			walkFiles(c, rel, fn)
			continue
		}
		fn(rel, c)
	}
}
