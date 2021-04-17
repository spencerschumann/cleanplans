package cleaner

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Color struct {
	R float64
	G float64
	B float64
}

func (n *SVGXMLNode) Style(name string) string {
	if n.style == nil {
		n.style = map[string]string{}
		n.styleNameOrder = map[string]int{}
		index := 0
		for _, pair := range strings.Split(n.Styles, ";") {
			kv := strings.Split(pair, ":")
			if len(kv) == 2 {
				n.style[kv[0]] = kv[1]
				index++
				n.styleNameOrder[kv[0]] = index
			} else {
				// error; unlikely. Ignore for now, but may want to warn.
			}
		}
	}
	return n.style[name]
}

func (n *SVGXMLNode) SetStyle(name string, value string) {
	if n.style == nil {
		// Call for side-effect of populating the style map
		n.Style(name)
	}
	n.style[name] = value
}

func (n *SVGXMLNode) RemoveStyle(name string) {
	delete(n.style, name)
}

func (n *SVGXMLNode) serializeStyle() {
	if n.style == nil {
		return
	}
	type nameValue struct {
		name  string
		value string
	}
	var styles []nameValue
	for name, value := range n.style {
		styles = append(styles, nameValue{name: name, value: value})
	}
	sort.Slice(styles, func(i, j int) bool {
		a := styles[i].name
		b := styles[j].name
		ao := n.styleNameOrder[a]
		bo := n.styleNameOrder[b]
		if ao == 0 || bo == 0 {
			return a < b
		}
		return ao < bo
	})
	var styleStrs []string
	for _, style := range styles {
		styleStrs = append(styleStrs, style.name+":"+style.value)
	}
	n.Styles = strings.Join(styleStrs, ";")
}

var rgbPercentRE = regexp.MustCompile(`rgb\(([0-9.]+)%,([0-9.]+)%,([0-9.]+)%\)`)
var hexColorRE = regexp.MustCompile(`#([[:xdigit:]]{2})([[:xdigit:]]{2})([[:xdigit:]]{2})`)

func parseColor(color string) (Color, error) {
	rgb := rgbPercentRE.FindStringSubmatch(color)
	if rgb != nil {
		return Color{
			R: ParseNumber(rgb[1]) / 100,
			G: ParseNumber(rgb[2]) / 100,
			B: ParseNumber(rgb[3]) / 100,
		}, nil
	}

	hex := hexColorRE.FindStringSubmatch(color)
	if hex != nil {
		parse := func(channel string) float64 {
			val, _ := strconv.ParseUint(channel, 16, 64)
			return float64(val) / 255.0
		}
		return Color{
			R: parse(hex[1]),
			G: parse(hex[2]),
			B: parse(hex[3]),
		}, nil
	}

	return Color{}, fmt.Errorf("unknown color description %q", color)
}
