package cords

import (
	"fmt"
	"io"
)

type nodeids struct {
	idTable map[*cordNode]int
	max     int
}

func newtable() nodeids {
	return nodeids{
		idTable: make(map[*cordNode]int),
		max:     1,
	}
}

func (ids nodeids) find(node *cordNode) int {
	return ids.idTable[node]
}

func (ids *nodeids) alloc(node *cordNode) int {
	if id := ids.find(node); id > 0 {
		return id
	}
	ids.idTable[node] = ids.max
	ids.max++
	return ids.max - 1
}

// Cord2Dot outputs the internal structure of a Cord in Graphviz DOT format
// (for debugging purposes).
//
func Cord2Dot(text Cord, w io.Writer) {
	io.WriteString(w, "strict digraph {\n")
	io.WriteString(w, "\tnode [fontname=Arial,fontsize=12];\n")
	ids := newtable()
	nodelist, edgelist := "", ""
	err := text.each(func(node *cordNode, pos uint64, depth int) error {
		ID := ids.alloc(node)
		styles := nodeDotStyles(node, node.IsLeaf(), false)
		if node.IsLeaf() {
			leaf := node.AsLeaf()
			strstart(leaf)
			label := fmt.Sprintf("%d @%d\\n“%s”", node.Weight(), pos, strstart(leaf))
			nodelist += fmt.Sprintf("\"%d\" [label=\"%s\" %s];\n", ID, label, styles)
		} else {
			inner := node.AsInner()
			if inner.Left() == nil {
				nilid := ID + 10000
				nodelist += fmt.Sprintf("\"%d\" %s;\n", nilid, emptyNode(nilid))
				edgelist += fmt.Sprintf("\"%d\" -> \"%d\";\n", ID, nilid)
			} else {
				edgelist += fmt.Sprintf("\"%d\" -> \"%d\";\n", ID, ids.find(inner.left))
			}
			if inner.Right() == nil {
				nilid := ID + 10000
				nodelist += fmt.Sprintf("\"%d\" %s;\n", nilid, emptyNode(nilid))
				edgelist += fmt.Sprintf("\"%d\" -> \"%d\";\n", ID, nilid)
			} else {
				_ = ids.alloc(inner.right)
				edgelist += fmt.Sprintf("\"%d\" -> \"%d\";\n", ID, ids.find(inner.right))
			}
			nodelist += fmt.Sprintf("\"%d\" [label=%d %s];\n", ID, node.Weight(), styles)
		}
		return nil
	})
	if err != nil {
		T().Errorf("cord DOT: %s", err.Error())
	}
	io.WriteString(w, nodelist)
	io.WriteString(w, edgelist)
	io.WriteString(w, "}\n")
}

func emptyNode(id int) string {
	s := "[label=\"\",color=black,shape=circle,fixedsize=true,width=.4]"
	//s = fmt.Sprintf(s, id)
	return s
}

func nodeDotStyles(node *cordNode, isleaf bool, highlight bool) string {
	s := ",style=filled"
	if isleaf {
		//s += ",fillcolor=\"#a3d7e4\""
		s += ",shape=box"
	} else {
		s += ",color=black,fillcolor=\"#a3d7e4\""
		s += ",shape=circle"
	}
	// if highlight {
	// 	s = s + fmt.Sprintf(",fillcolor=\"%s\"", hexhlcolors[node.pathcnt])
	// } else {
	// 	s = s + fmt.Sprintf(",fillcolor=\"%s\"", hexcolors[node.pathcnt])
	// }
	return s
}

var hexhlcolors = [...]string{"#FFEEDD", "#FFDDCC", "#FFCCAA", "#FFBB88", "#FFAA66",
	"#FF9944", "#FF8822", "#FF7700", "#ff6600"}

var hexcolors = [...]string{"white", "#CCDDFF", "#AACCFF", "#88BBFF", "#66AAFF",
	"#4499FF", "#2288FF", "#0077FF", "#0066FF"}
