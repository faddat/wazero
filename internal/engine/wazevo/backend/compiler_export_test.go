package backend

// ExportLower is a wrapper of lower() for testing.
func ExportLower(cc Compiler) {
	c := cc.(*compiler)
	c.lower()
}
