// Copyright ©2017,2019,2020 Steve Merrony

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

type cell struct {
	charValue                                byte
	blink, dim, reverse, underscore, protect bool
}

func (c *cell) set(cv byte, bl, dm, rev, under, prot bool) {
	c.charValue = cv
	c.blink = bl
	c.dim = dm
	c.reverse = rev
	c.underscore = under
	c.protect = prot
}

func (c *cell) clearToSpace() {
	c.charValue = ' '
	c.blink = false
	c.dim = false
	c.reverse = false
	c.underscore = false
	c.protect = false
}

func (c *cell) clearToSpaceIfUnprotected() {
	if !c.protect {
		c.clearToSpace()
	}
}

func (c *cell) copyFrom(src cell) {
	c.charValue = src.charValue
	c.blink = src.blink
	c.dim = src.dim
	c.reverse = src.reverse
	c.underscore = src.underscore
	c.protect = src.protect
}
