package expr

// CollectIdentifiers compiles src and returns the deduplicated set of
// identifier names referenced in the expression.
//
// Rules:
//   - Identifier → collect the name.
//   - MemberExpr with Computed=false (dot notation) → walk Object only.
//   - MemberExpr with Computed=true (bracket notation) → walk Object and Property.
//   - UnaryExpr → walk Operand.
//   - BinaryExpr → walk Left and Right.
//   - TernaryExpr → walk Condition, Consequent, and Alternate.
//   - CallExpr → walk Callee and all Args.
//   - ArrayLit → walk all Elements.
//   - ObjectLit → walk all property Value nodes.
//   - Literal nodes → skip.
func CollectIdentifiers(src string) ([]string, error) {
	tokens, err := Tokenize(src)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	node, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	collectFromNode(node, seen)

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	return result, nil
}

func collectFromNode(node Node, seen map[string]bool) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *Identifier:
		seen[n.Name] = true
	case *MemberExpr:
		collectFromNode(n.Object, seen)
		if n.Computed {
			collectFromNode(n.Property, seen)
		}
	case *UnaryExpr:
		collectFromNode(n.Operand, seen)
	case *BinaryExpr:
		collectFromNode(n.Left, seen)
		collectFromNode(n.Right, seen)
	case *TernaryExpr:
		collectFromNode(n.Condition, seen)
		collectFromNode(n.Consequent, seen)
		collectFromNode(n.Alternate, seen)
	case *CallExpr:
		collectFromNode(n.Callee, seen)
		for _, arg := range n.Args {
			collectFromNode(arg, seen)
		}
	case *ArrayLit:
		for _, elem := range n.Elements {
			collectFromNode(elem, seen)
		}
	case *ObjectLit:
		for _, prop := range n.Properties {
			collectFromNode(prop.Value, seen)
		}
	// Literal nodes: NumberLit, StringLit, BoolLit, NullLit, UndefinedLit → skip
	}
}
