package cords

// Concat concatenates two cords to a new one
func Concat(c1, c2 Cord) Cord {
	if c1.IsVoid() {
		return c2
	}
	if c2.IsVoid() {
		return c1
	}
	r := makeInnerNode()
	r.weight = c1.Len() + c2.Len()
	r.left = &c1.root.cordNode
	c1.root.right = c2.root.left
	return Cord{root: r}
}
