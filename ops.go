package cords

// Concat concatenates two cords to a new one
func Concat(c1, c2 Cord) Cord {
	if c1.root == nil {
		return c2
	}
	if c2.root == nil {
		return c1
	}
	r := cordNode{weight: c1.Len() + c2.Len()}
	r.left = c1.root
	c1.root.right = c2.root.left
	return Cord{root: &r}
}
