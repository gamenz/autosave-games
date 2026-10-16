package config

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const scanHeader = "\n# Steam scan: games without a known save folder.\n" +
	"# Fill in \"saves\" and remove the leading \"# \" from a game's lines to enable it.\n"

var emptyGamesLine = regexp.MustCompile(`(?m)^games: *(\[\]|null|~)? *$`)

// mergeYAML appends added games to the games sequence (moved to the end of
// the document) and needSaves games as commented blocks after it.
func mergeYAML(text string, added, needSaves []NewGame) (string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("parse config: top level is not a mapping")
	}
	root := doc.Content[0]
	games := takeGamesPair(root)
	for _, g := range added {
		games.Content = append(games.Content, gameNode(g))
	}
	out, err := encode(&doc)
	if err != nil {
		return "", err
	}
	if len(needSaves) == 0 {
		return out, nil
	}
	// Commented items must follow a block "games:" key to be valid once enabled.
	if len(games.Content) == 0 {
		out = emptyGamesLine.ReplaceAllString(out, "games:")
	}
	var b strings.Builder
	b.WriteString(strings.TrimRight(out, "\n") + "\n" + scanHeader)
	for _, g := range needSaves {
		block, err := commentedBlock(g)
		if err != nil {
			return "", err
		}
		b.WriteString(block)
	}
	return b.String(), nil
}

// takeGamesPair moves the games key/value to the end of the root mapping
// (creating it if missing) and returns the sequence node in block style.
func takeGamesPair(root *yaml.Node) *yaml.Node {
	key, value := scalar("games", 0), &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "games" {
			key, value = root.Content[i], root.Content[i+1]
			root.Content = append(root.Content[:i], root.Content[i+2:]...)
			break
		}
	}
	if value.Kind != yaml.SequenceNode {
		value = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	}
	value.Style &^= yaml.FlowStyle
	root.Content = append(root.Content, key, value)
	return value
}

func encode(n *yaml.Node) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}
	return buf.String(), nil
}

// commentedBlock renders g as a disabled games item with its marker line.
func commentedBlock(g NewGame) (string, error) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{gameNode(g)}}
	text, err := encode(seq)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "#   # steam_appid %d: save path unknown\n", g.SteamAppID)
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		b.WriteString("#   " + line + "\n")
	}
	return b.String(), nil
}

func gameNode(g NewGame) *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	add := func(k string, v *yaml.Node) { m.Content = append(m.Content, scalar(k, 0), v) }
	add("name", scalar(g.Name, 0))
	if g.SteamAppID != 0 {
		add("steam_appid", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprint(g.SteamAppID)})
	}
	if g.InstallDir != "" {
		add("install_dir", scalar(g.InstallDir, yaml.SingleQuotedStyle))
	}
	add("processes", flowSeq(g.Processes, 0))
	add("saves", flowSeq(g.Saves, yaml.SingleQuotedStyle))
	return m
}

func scalar(v string, style yaml.Style) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v, Style: style}
}

func flowSeq(items []string, style yaml.Style) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
	for _, it := range items {
		n.Content = append(n.Content, scalar(it, style))
	}
	return n
}
